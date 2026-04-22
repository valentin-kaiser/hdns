import { HttpClient } from '@angular/common/http';
import { Injectable, Signal, signal } from '@angular/core';
import {
  Observable,
  Subject,
  catchError,
  defer,
  finalize,
  repeat,
  retry,
  shareReplay,
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
  Empty,
  HDNSDefinition,
  Record,
  RecordDelete,
  RecordList,
  Request,
  Resolution,
  ResolutionResult,
  ZoneList,
  ZoneRequest,
} from '../../model/api';
import { LoggerService } from '../logger/logger.service';

export interface Stream<TOut, TIn> {
  messages$: Observable<TOut>;
  terminate$: Observable<CloseEvent | 'client'>;
  connect$: Observable<void>;
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

  public getZones(req: ZoneRequest): Observable<ZoneList> {
    return this.rpc<ZoneList>('getZones', req);
  }

  public getRecords(req: Request): Observable<RecordList> {
    return this.rpc<RecordList>('getRecords', req);
  }

  public upsertRecord(record: Record): Observable<Record> {
    return this.rpc<Record>('upsertRecord', record);
  }

  public deleteRecord(record: Record, deleteFromHetzner: boolean): Observable<Empty> {
    const req: RecordDelete = { record, deleteFromHetzner };
    return this.rpc<Empty>('deleteRecord', req);
  }

  public refreshRecord(record: Record): Observable<Record> {
    return this.rpc<Record>('refreshRecord', record);
  }

  public resolveRecord(record: Record): Observable<ResolutionResult> {
    return this.rpc<ResolutionResult>('resolveRecord', record);
  }

  public streamResolveRecord(): Stream<Resolution, Record> {
    return this.stream<Resolution, Record>('streamResolveRecord');
  }

  public getAddress(): Observable<Address> {
    return this.rpc<Address>('getAddress', {});
  }

  public streamAddress(): Stream<Address, Empty> {
    return this.stream<Address, Empty>('streamAddress');
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
    params?: any,
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

    const method = HDNSDefinition.methods[methodKey];
    const path = `/rpc/${HDNSDefinition.name}/${method.name}`;
    const url = this.toWebsocketURL(path, { ...params });

    let currentWs: WebSocketSubject<any> | null = null;
    const outgoing$ = new Subject<TIn>();
    const terminate$ = new Subject<CloseEvent | 'client'>();
    const kill$ = new Subject<void>();
    const connectSource$ = new Subject<void>();
    const connect$ = connectSource$.pipe(shareReplay({ bufferSize: 1, refCount: true }));

    this.logger.info(`${this.logType} ${this.logName} - ${method.name} WS connect`);

    // One WS per subscription; we reconnect by re-subscribing via retryWhen
    const source$ = defer(() => {
      const config: WebSocketSubjectConfig<any> = {
        url,
        deserializer: (e: MessageEvent) => JSON.parse(e.data), // -> TOut
        serializer: (value: any) => JSON.stringify(value), // <- TIn
        openObserver: {
          next: () => {
            this.logger.info(`${this.logType} ${this.logName} WS open ${url}`);
            connectSource$.next();
          },
        },
        closeObserver: {
          next: (ev: CloseEvent) => {
            this.logger.info(
              `${this.logType} ${this.logName} - ${method.name} WS closed code=${ev.code} reason=${ev.reason}`,
            );
            terminate$.next(ev);
          },
        },
      };

      const ws = webSocket<any>(config);
      currentWs = ws;

      // Pipe queued outgoing messages into the active socket
      const outSub = outgoing$.subscribe({
        next: (value) => {
          try {
            ws.next(value);
          } catch (err) {
            this.logger.warn(
              `${this.logType} ${this.logName} - ${method.name} WS send failed (will retry on reconnect)`,
              err,
            );
            // If send fails due to closed socket, the retryWhen will recreate ws.
          }
        },
      });

      // When this WS completes/errors, stop feeding it
      return ws.pipe(
        finalize(() => {
          outSub.unsubscribe();
          currentWs = null;
        }),
      );
    });

    const messages$ = source$
      .pipe(
        reconnect
          ? retry({
              count: maxRetries,
              resetOnSuccess: true,
              delay: (error, retryCount) => {
                const backoff = Math.min(
                  maxBackoffMs,
                  Math.floor(backoffMs * Math.pow(2, Math.max(0, retryCount - 1))),
                );
                const jitter = Math.floor(Math.random() * 300);
                this.logger.warn(
                  `${this.logType} ${this.logName} - ${method.name} WS retry #${retryCount} in ${backoff + jitter}ms`,
                  error,
                );
                return timer(backoff + jitter);
              },
            })
          : tap({}),
        // Also re-connect when the server closes the socket cleanly (stream completes
        // without an error). `retry` only handles errors, so add `repeat` for
        // completions with the same exponential backoff.
        reconnect
          ? repeat({
              count: maxRetries,
              delay: (repeatCount) => {
                const backoff = Math.min(
                  maxBackoffMs,
                  Math.floor(backoffMs * Math.pow(2, Math.max(0, repeatCount - 1))),
                );
                const jitter = Math.floor(Math.random() * 300);
                this.logger.warn(
                  `${this.logType} ${this.logName} - ${method.name} WS reconnect #${repeatCount} in ${backoff + jitter}ms (server closed stream)`,
                );
                return timer(backoff + jitter);
              },
            })
          : tap({}),
        takeUntil(kill$),
        shareReplay({ bufferSize: 1, refCount: true }),
      )
      .pipe(
        tap({
          next: (msg) =>
            this.logger.debug(
              `${this.logType} ${this.logName} - ${method.name} WS message received`,
              msg,
            ),
          error: (err) =>
            this.logger.error(
              `${this.logType} ${this.logName} - ${method.name} WS message error`,
              err,
            ),
        }),
      ) as Observable<TOut>;

    const send = (msg: TIn) => {
      this.logger.debug(`${this.logType} ${this.logName} - ${method.name} WS message sent`, msg);
      outgoing$.next(msg);
    };

    const close = () => {
      // Stop retries and close current socket
      kill$.next();
      kill$.complete();
      try {
        currentWs?.complete();
      } catch {
        /* ignore */
      }
      outgoing$.complete();
      connectSource$.complete();
      terminate$.next('client');
      terminate$.complete();
      this.logger.info(`${this.logType} ${this.logName} - ${method.name} WS closed by client`);
    };

    return { messages$, terminate$, connect$, send, close };
  }

  // ── Private helpers ───────────────────────────────────────────────────────

  private buildURL(): void {
    if (!environment.production) {
      return;
    }
    const url = new URL(window.location.href);
    this.baseURL = url.protocol + '//' + url.hostname + (url.port ? `:${url.port}` : '');
  }

  private toWebsocketURL(endpoint: string, params?: any): string {
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
