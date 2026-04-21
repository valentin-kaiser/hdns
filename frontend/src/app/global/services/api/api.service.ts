import { HttpClient } from '@angular/common/http';
import { Injectable, Signal, signal } from '@angular/core';
import {
  Observable,
  Subject,
  catchError,
  defer,
  finalize,
  retry,
  takeUntil,
  tap,
  throwError,
  timer,
} from 'rxjs';
import { WebSocketSubject, WebSocketSubjectConfig, webSocket } from 'rxjs/webSocket';
import { environment } from '../../../../environments/environment';
import {
  Address,
  AddressHistory,
  Configuration,
  Record as DnsRecord,
  Empty,
  HDNSDefinition,
  RecordDelete,
  RecordList,
  Request,
  ResolutionResult,
  ZoneList,
} from '../../model/api';
import { LoggerService } from '../logger/logger.service';

export interface Stream<TOut, TIn> {
  messages: Signal<TOut[]>;
  latestMessage: Signal<TOut | null>;
  isConnected: Signal<boolean>;
  terminateEvent: Signal<CloseEvent | 'client' | null>;
  error: Signal<any>;
  send: (msg: TIn) => void;
  close: () => void;
}

export interface ApiCallResult<T> {
  data: Signal<T | null>;
  loading: Signal<boolean>;
  error: Signal<any>;
}

@Injectable({ providedIn: 'root' })
export class ApiService {
  private readonly logType = '[Service]';
  private readonly logName = '[Api]';
  private baseURL = 'https://localhost:443';

  private readonly _initialized = signal(false);
  readonly initialized: Signal<boolean> = this._initialized.asReadonly();

  constructor(
    private readonly logger: LoggerService,
    private readonly http: HttpClient,
  ) {
    this.logger.info(`${this.logType} ${this.logName} constructor`);
    this.buildURL();
  }

  public getZones(req: Request): Observable<ZoneList> {
    return this.rpc<ZoneList>('getZones', req);
  }

  public getRecords(): Observable<RecordList> {
    return this.rpc<RecordList>('getRecords', {});
  }

  public upsertRecord(record: DnsRecord): Observable<DnsRecord> {
    return this.rpc<DnsRecord>('upsertRecord', record);
  }

  public deleteRecord(record: DnsRecord, deleteFromHetzner: boolean): Observable<Empty> {
    const req: RecordDelete = { record, deleteFromHetzner };
    return this.rpc<Empty>('deleteRecord', req);
  }

  public refreshRecord(record: DnsRecord): Observable<DnsRecord> {
    return this.rpc<DnsRecord>('refreshRecord', record);
  }

  public resolveRecord(record: DnsRecord): Observable<ResolutionResult> {
    return this.rpc<ResolutionResult>('resolveRecord', record);
  }

  public streamResolveRecord(record: DnsRecord): Stream<ResolutionResult, unknown> {
    return this.stream<ResolutionResult, unknown>('streamResolveRecord', { ...record });
  }

  public getAddress(): Observable<Address> {
    return this.rpc<Address>('getAddress', {});
  }

  public streamAddress(): Stream<Address, unknown> {
    return this.stream<Address, unknown>('streamAddress');
  }

  public getAddressHistory(): Observable<AddressHistory> {
    return this.rpc<AddressHistory>('getAddressHistory', {});
  }

  public refreshAddress(): Observable<Address> {
    return this.rpc<Address>('refreshAddress', {});
  }

  public getConfig(): Observable<Configuration> {
    return this.rpc<Configuration>('getConfig', {});
  }

  public updateConfig(config: Configuration): Observable<Configuration> {
    return this.rpc<Configuration>('updateConfig', config);
  }

  public streamLogs(): Stream<any, unknown> {
    return this.stream<any, unknown>('streamLogs');
  }

  /**
   * Generic RPC method. All API calls go through this, which ensures consistent logging and error handling.
   */
  private rpc<T>(methodKey: keyof typeof HDNSDefinition.methods, body: unknown): Observable<T> {
    const method = HDNSDefinition.methods[methodKey];
    const path = `/rpc/${HDNSDefinition.name}/${method.name}`;
    return this.post<T>(path, body);
  }

  /**
   * POST request. Token is attached by the authInterceptor registered in app.config.ts.
   * On 401 the interceptor clears the session and redirects to login.
   */
  post<T = any>(path: string, body: unknown): Observable<T> {
    const url = this.baseURL + path;
    this.logger.info(`${this.logType} ${this.logName} POST ${url}`, body);
    return this.http.post<T>(url, body).pipe(
      tap((res) => this.logger.info(`${this.logType} ${this.logName} ${path} response`, res)),
      catchError((err) => {
        this.logger.error(`${this.logType} ${this.logName} ${path} error`, err);
        if (err.status <= 0) {
          return throwError(() => {
            return {
              error:
                'Unable to connect to the server. Please check your network connection and try again.',
            };
          });
        }
        return throwError(() => err);
      }),
    );
  }

  // ── WebSocket ─────────────────────────────────────────────────────────────

  public stream<TOut, TIn = unknown>(
    methodKey: keyof typeof HDNSDefinition.methods,
    params?: Record<string, any>,
    opts: {
      reconnect?: boolean;
      maxRetries?: number;
      backoffMs?: number;
      maxBackoffMs?: number;
    } = {},
  ): Stream<TOut, TIn> {
    const {
      reconnect = true,
      maxRetries = Infinity,
      backoffMs = 1000,
      maxBackoffMs = 10000,
    } = opts;

    let currentWs: WebSocketSubject<any> | null = null;
    const outgoing$ = new Subject<TIn>();
    const kill$ = new Subject<void>();

    const messagesSignal = signal<TOut[]>([]);
    const latestMessageSignal = signal<TOut | null>(null);
    const isConnectedSignal = signal<boolean>(false);
    const terminateEventSignal = signal<CloseEvent | 'client' | null>(null);
    const errorSignal = signal<any>(null);

    const method = HDNSDefinition.methods[methodKey];
    const path = `/rpc/${HDNSDefinition.name}/${method.name}`;
    const url = this.toWebsocketURL(path, { ...params });

    const source$ = defer(() => {
      this.logger.info(`${this.logType} ${this.logName} connecting to WebSocket ${url}`);
      const config: WebSocketSubjectConfig<any> = {
        url,
        deserializer: (e: MessageEvent) => JSON.parse(e.data),
        serializer: (v: any) => JSON.stringify(v),
        openObserver: { next: () => isConnectedSignal.set(true) },
        closeObserver: {
          next: (ev: CloseEvent) => {
            isConnectedSignal.set(false);
            terminateEventSignal.set(ev);
          },
        },
      };
      currentWs = webSocket<any>(config);
      const outSub = outgoing$.subscribe({
        next: (v) => {
          this.logger.debug(`${this.logType} ${this.logName} [stream:${method.name}] >>> sent:`, v);
          try {
            currentWs!.next(v);
          } catch (e) {
            errorSignal.set(e);
          }
        },
      });
      return currentWs.pipe(
        finalize(() => {
          outSub.unsubscribe();
          currentWs = null;
          isConnectedSignal.set(false);
        }),
      );
    });

    const subscription = source$
      .pipe(
        reconnect
          ? retry({
              count: maxRetries,
              resetOnSuccess: true,
              delay: (err, n) => {
                const ms =
                  Math.min(maxBackoffMs, backoffMs * Math.pow(2, Math.max(0, n - 1))) +
                  Math.random() * 300;
                errorSignal.set(err);
                return timer(ms);
              },
            })
          : tap({}),
        takeUntil(kill$),
      )
      .subscribe({
        next: (msg: TOut) => {
          this.logger.debug(
            `${this.logType} ${this.logName} [stream:${method.name}] <<< received:`,
            msg,
          );
          messagesSignal.update((m) => [...m, msg]);
          latestMessageSignal.set(msg);
        },
        error: (err) => {
          errorSignal.set(err);
          isConnectedSignal.set(false);
        },
        complete: () => isConnectedSignal.set(false),
      });

    this.logger.info(`${this.logType} ${this.logName} initialized WebSocket stream for ${url}`);

    return {
      messages: messagesSignal.asReadonly(),
      latestMessage: latestMessageSignal.asReadonly(),
      isConnected: isConnectedSignal.asReadonly(),
      terminateEvent: terminateEventSignal.asReadonly(),
      error: errorSignal.asReadonly(),
      send: (msg) => outgoing$.next(msg),
      close: () => {
        kill$.next();
        kill$.complete();
        try {
          currentWs?.complete();
          subscription.unsubscribe();
        } catch {
          /* ignore */
        }
        outgoing$.complete();
        terminateEventSignal.set('client');
        isConnectedSignal.set(false);
      },
    };
  }

  // ── Private helpers ───────────────────────────────────────────────────────

  private buildURL(): void {
    if (!environment.production) {
      return;
    }
    const url = new URL(window.location.href);
    this.baseURL = url.protocol + '//' + url.hostname + (url.port ? `:${url.port}` : '');
  }

  private toWebsocketURL(endpoint: string, params?: Record<string, any>): string {
    const base = new URL(this.baseURL);
    const url = new URL(endpoint, base);
    if (params) {
      Object.entries(params)
        .filter(([, v]) => v != null)
        .forEach(([k, v]) => url.searchParams.append(k, String(v)));
    }
    url.protocol = base.protocol === 'https:' ? 'wss:' : 'ws:';
    return url.toString();
  }
}
