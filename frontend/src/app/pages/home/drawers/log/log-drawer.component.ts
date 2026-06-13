import { CommonModule } from '@angular/common';
import { Component, OnDestroy, OnInit, signal, ViewChild } from '@angular/core';
import { Subscription } from 'rxjs';
import { DrawerComponent } from '../../../../components/drawer/drawer.component';
import { Empty, Line } from '../../../../global/model/api';
import { ApiService, Stream } from '../../../../global/services/api/api.service';
import { NotifyService } from '../../../../global/services/notify/notify.service';

interface LogField {
  key: string;
  value: string;
}

interface LogViewLine {
  isJson: boolean;
  level: string;
  levelClass: string;
  time: string;
  message: string;
  knownFields: LogField[];
  extraFields: LogField[];
  raw: string;
}

@Component({
  selector: 'app-log-drawer',
  standalone: true,
  imports: [
    CommonModule,
    DrawerComponent,
  ],
  template: `
    <app-drawer
      #drawer
      (closed)="drawer.close()"
      [width]="60"
      [breakpoints]="[
        { maxWidth: 768, width: 100 },
        { maxWidth: 1200, width: 80 },
        { maxWidth: 1600, width: 60 },
      ]"
    >
      <div class="drawer-header" header>
        <h3 class="drawer-title">Log</h3>
      </div>

      <div class="drawer-body" content>
        @for (line of lines(); track $index) {
          <div class="log-line" [class.log-line-malformed]="!line.isJson">
            <span class="log-time">[{{ line.time }}]</span>
            <span class="log-level" [class]="'log-level ' + line.levelClass">{{ line.level }}</span>
            <span class="log-message">{{ line.message }}</span>

            @for (field of line.knownFields; track field.key) {
              <span class="log-detail">
                <span class="detail-key">{{ field.key }}=</span>
                <span class="detail-value mono">{{ field.value }}</span>
              </span>
            }

            @for (field of line.extraFields; track field.key) {
              <span class="log-detail">
                <span class="detail-key">{{ field.key }}=</span>
                <span class="detail-value mono">{{ field.value }}</span>
              </span>
            }

            @if (!line.isJson) {
              <span class="log-detail raw-detail">
                <span class="detail-key">raw=</span>
                <span class="detail-value mono">{{ line.raw }}</span>
              </span>
            }
          </div>
        }
      </div>
    </app-drawer>
  `,
  styles: [
    `
      :host {
        display: flex;
        flex: 1;
        min-height: 0;
        flex-direction: column;
      }
      .drawer-header {
        padding: 20px 24px 16px;
        border-bottom: 1px solid var(--launch-border-color);
      }
      .drawer-title {
        margin: 0;
        font-size: 1rem;
        font-weight: 600;
        color: var(--launch-text-primary);
      }
      .drawer-body {
        flex: 1;
        min-height: 0;
        overflow-y: auto;
        padding: 20px 24px;
      }
      .loading-msg {
        color: var(--launch-text-muted);
        font-size: 0.875rem;
      }
      .log-line {
        font-family: 'Roboto Mono', monospace;
        font-size: 0.8125rem;
        line-height: 1.45;
        color: var(--launch-text-primary);
        white-space: normal;
        word-break: break-word;
        overflow-wrap: anywhere;
        padding: 2px 0;
      }
      .log-line + .log-line {
        border-top: 1px solid rgba(255, 255, 255, 0.03);
      }
      .log-line-malformed {
        color: var(--hdns-danger);
      }
      .log-level {
        font-weight: 700;
        margin: 0 0.5ch;
      }
      .level-trace,
      .level-debug {
        color: var(--launch-text-secondary);
      }
      .level-info {
        color: var(--hdns-info);
      }
      .level-warning,
      .level-warn {
        color: var(--hdns-warning);
      }
      .level-error {
        color: var(--hdns-danger);
      }
      .log-time {
        font-size: 0.75rem;
        color: var(--launch-text-muted);
        font-variant-numeric: tabular-nums;
        margin-right: 0.5ch;
      }
      .log-message {
        font-size: 0.8125rem;
        color: var(--launch-text-primary);
        margin-right: 0.5ch;
      }
      .log-detail {
        display: inline-flex;
        align-items: baseline;
        gap: 0;
        margin-right: 0.75ch;
      }
      .detail-key {
        color: var(--hdns-info);
        font-size: 0.6875rem;
      }
      .detail-value {
        color: var(--launch-text-primary);
        font-size: 0.75rem;
        overflow-wrap: anywhere;
        word-break: break-word;
      }
      .mono {
        font-family: 'Roboto Mono', monospace;
        font-variant-numeric: tabular-nums;
      }
      .raw-detail .detail-key,
      .raw-detail .detail-value {
        color: var(--hdns-danger);
      }
    `,
  ],
})
export class LogDrawerComponent implements OnInit, OnDestroy {
  @ViewChild('drawer') drawer!: DrawerComponent;

  stream: Stream<Line, Empty> | null = null;
  private messagesSub?: Subscription;
  lines = signal<LogViewLine[]>([]);
  private readonly maxLines = 800;

  constructor(
    private readonly api: ApiService,
    private readonly notify: NotifyService,
  ) {}

  ngOnInit(): void {}

  ngOnDestroy(): void {
    this.cleanupStream();
  }

  open() {
    this.lines.set([]);
    this.drawer.open();

    this.cleanupStream();

    this.stream = this.api.streamLog();
    this.messagesSub = this.stream.messages$.subscribe({
      next: (entry) => {
        const parsed = this.parseLogLine(entry.line ?? '');
        this.lines.update((lines) => [parsed, ...lines].slice(0, this.maxLines));
      },
      error: (err) => {
        this.notify.error('Failed to stream log', err);
        this.drawer.close();
      },
    });
    this.stream.send({});
  }

  close() {
    this.drawer.close();
    this.lines.set([]);
    this.cleanupStream();
  }

  private cleanupStream(): void {
    if (this.messagesSub) {
      this.messagesSub.unsubscribe();
      this.messagesSub = undefined;
    }

    if (this.stream) {
      this.stream.close();
      this.stream = null;
    }
  }

  private parseLogLine(rawLine: string): LogViewLine {
    let parsedData: Record<string, unknown> | null = null;

    try {
      const candidate = JSON.parse(rawLine) as unknown;
      if (candidate && typeof candidate === 'object' && !Array.isArray(candidate)) {
        parsedData = candidate as Record<string, unknown>;
      }
    } catch {
      parsedData = null;
    }

    if (!parsedData) {
      return {
        isJson: false,
        level: 'RAW',
        levelClass: 'level-error',
        time: this.formatTime(''),
        message: 'Unable to parse log line JSON',
        knownFields: [],
        extraFields: [],
        raw: rawLine,
      };
    }

    const levelRaw = this.readString(parsedData['level']) || 'unknown';
    const level = levelRaw.toUpperCase();
    const timeRaw = this.readString(parsedData['time']);
    const message = this.readString(parsedData['message']) || '(no message)';

    const knownFieldOrder = ['caller', 'package', 'remote', 'method', 'duration'];
    const knownFields: LogField[] = knownFieldOrder
      .filter((key) => key in parsedData)
      .map((key) => ({ key, value: this.formatValue(parsedData[key]) }));

    const consumedKeys = new Set(['time', 'level', 'message', ...knownFieldOrder]);
    const extraFields: LogField[] = Object.keys(parsedData)
      .filter((key) => !consumedKeys.has(key))
      .sort((a, b) => a.localeCompare(b))
      .map((key) => ({ key, value: this.formatValue(parsedData[key]) }));

    return {
      isJson: true,
      level,
      levelClass: this.levelClass(levelRaw),
      time: this.formatTime(timeRaw),
      message,
      knownFields,
      extraFields,
      raw: rawLine,
    };
  }

  private levelClass(level: string): string {
    switch (level.toLowerCase()) {
      case 'trace':
        return 'level-trace';
      case 'debug':
        return 'level-debug';
      case 'info':
        return 'level-info';
      case 'warn':
      case 'warning':
        return 'level-warning';
      case 'error':
      case 'fatal':
      case 'panic':
        return 'level-error';
      default:
        return 'level-debug';
    }
  }

  private readString(value: unknown): string {
    if (typeof value === 'string') return value;
    if (typeof value === 'number' || typeof value === 'boolean') return String(value);
    return '';
  }

  private formatValue(value: unknown): string {
    if (value === null) return 'null';
    if (typeof value === 'string') return value;
    if (typeof value === 'number' || typeof value === 'boolean') return String(value);

    try {
      return JSON.stringify(value);
    } catch {
      return String(value);
    }
  }

  private formatTime(value: string): string {
    if (!value) return 'Unknown time';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return value;
    return date.toLocaleTimeString();
  }
}
