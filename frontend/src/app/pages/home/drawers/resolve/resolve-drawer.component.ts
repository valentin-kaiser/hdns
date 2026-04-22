import { CommonModule } from '@angular/common';
import { ChangeDetectorRef, Component, OnChanges, OnDestroy, ViewChild } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { Subscription } from 'rxjs';
import { DrawerComponent } from '../../../../components/drawer/drawer.component';
import { Record as DnsRecord, Resolution } from '../../../../global/model/api';
import { ApiService, Stream } from '../../../../global/services/api/api.service';
import { NotifyService } from '../../../../global/services/notify/notify.service';

@Component({
  selector: 'app-resolve-drawer',
  standalone: true,
  imports: [CommonModule, MatIconModule, MatButtonModule, MatProgressSpinnerModule, DrawerComponent],
  template: `
    <app-drawer #drawer (closed)="drawer.close()" [width]="45" [breakpoints]="[{ maxWidth: 768, width: 100 }, { maxWidth: 1200, width: 80 }, { maxWidth: 1600, width: 60 }]">
    <div class="drawer-header" header>
      <h3 class="drawer-title">Resolve: {{ record?.name }}.{{ record?.domain }}</h3>
    </div>

    <div class="drawer-body" content>

      @if (results.size === 0) {
        <div class="empty-state">
          <mat-icon class="empty-icon">travel_explore</mat-icon>
          <div class="empty-title">Waiting for results</div>
          <div class="empty-sub">DNS servers will appear here as they respond.</div>
        </div>
      }

      @if (results.size > 0) {
        <div class="result-list">
          @for (r of results.values(); track r.server) {
            <div class="result-row" [class.has-error]="!!r.error" [class.has-success]="!r.error && !!r.addresses?.length">
              <div class="server-col">
                <span class="status-dot"
                      [class.dot-ok]="!r.error && !!r.addresses?.length"
                      [class.dot-err]="!!r.error"></span>
                <div class="server-meta">
                  <div class="server-label">DNS Server</div>
                  <div class="server-name">{{ r.server }}</div>
                </div>
              </div>

              <div class="content-col">
                @if (r.responseTime) {
                  <div class="response-pill" [class.pill-err]="!!r.error">
                    <mat-icon class="pill-icon">schedule</mat-icon>
                    <span>{{ r.responseTime }}ms</span>
                  </div>
                }

                @if (r.addresses?.length && !r.error) {
                  <div class="address-list">
                    @for (a of r.addresses; track a) {
                      <div class="address-row">{{ a }}</div>
                    }
                  </div>
                }

                @if (r.error) {
                  <div class="result-error">
                    <mat-icon class="err-icon">error_outline</mat-icon>
                    <span>{{ r.error }}</span>
                  </div>
                }

                @if (!r.error && !r.addresses?.length) {
                  <div class="pending-msg">No addresses returned.</div>
                }
              </div>
            </div>
          }
        </div>
      }
    </div>

    <div class="drawer-footer" footer>
      <button mat-stroked-button (click)="close()">Close</button>
    </div>
  `,
  styles: [`
    :host {
      display: flex;
      flex: 1;
      min-height: 0;
      flex-direction: column;
    }
    .drawer-header { padding: 20px 24px 16px; border-bottom: 1px solid var(--launch-border-color); }
    .drawer-title { margin: 0; font-size: 1rem; font-weight: 600; color: var(--launch-text-primary); }
    .drawer-body {
      flex: 1;
      min-height: 0;
      overflow-y: auto;
      padding: 20px 24px;
      display: flex;
      flex-direction: column;
      gap: 12px;
    }
    .spinner-row { display: flex; align-items: center; gap: 10px; color: var(--launch-text-muted); font-size: 0.875rem; }

    .empty-state {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      gap: 6px;
      padding: 48px 16px;
      color: var(--launch-text-muted);
      text-align: center;
    }
    .empty-icon { font-size: 40px; width: 40px; height: 40px; opacity: 0.6; }
    .empty-title { font-size: 0.95rem; font-weight: 600; color: var(--launch-text-secondary); }
    .empty-sub { font-size: 0.8125rem; }

    .result-list { display: flex; flex-direction: column; gap: 10px; }

    .result-row {
      display: grid;
      grid-template-columns: 1fr;
      gap: 0;
      background: #16161e;
      border: 1px solid var(--launch-border-color);
      border-radius: 8px;
      overflow: hidden;
      transition: border-color 0.15s ease;
    }
    .result-row.has-success { border-color: color-mix(in srgb, var(--hdns-success) 40%, var(--launch-border-color)); }
    .result-row.has-error { border-color: color-mix(in srgb, var(--hdns-danger) 50%, var(--launch-border-color)); }

    .server-col {
      display: flex;
      align-items: center;
      gap: 10px;
      padding: 12px 14px;
      background: rgba(255, 255, 255, 0.02);
      border-right: 1px solid var(--launch-border-color);
      min-width: 0;
    }
    .status-dot {
      width: 8px; height: 8px;
      border-radius: 50%;
      background: var(--launch-text-muted);
      flex-shrink: 0;
      box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.04);
    }
    .status-dot.dot-ok { background: var(--hdns-success); box-shadow: 0 0 0 3px var(--hdns-primary-tint); }
    .status-dot.dot-err { background: var(--hdns-danger); box-shadow: 0 0 0 3px color-mix(in srgb, var(--hdns-danger) 20%, transparent); }
    .server-meta { min-width: 0; display: flex; flex-direction: column; gap: 2px; }
    .server-label {
      font-size: 0.6875rem;
      text-transform: uppercase;
      letter-spacing: 0.06em;
      color: var(--launch-text-muted);
      font-weight: 500;
    }
    .server-name {
      font-family: 'Roboto Mono', monospace;
      font-size: 0.8125rem;
      font-weight: 600;
      color: var(--launch-text-primary);
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .content-col {
      position: relative;
      padding: 12px 14px;
      display: flex;
      flex-direction: column;
      gap: 8px;
      min-width: 0;
    }

    .response-pill {
      position: absolute;
      top: 10px;
      right: 12px;
      display: inline-flex;
      align-items: center;
      gap: 4px;
      padding: 2px 8px;
      border-radius: 999px;
      background: rgba(255, 255, 255, 0.04);
      border: 1px solid var(--launch-border-color);
      font-size: 0.6875rem;
      color: var(--launch-text-muted);
      font-variant-numeric: tabular-nums;
    }
    .pill-icon { font-size: 12px; width: 12px; height: 12px; }

    .address-list {
      display: flex;
      flex-direction: column;
      gap: 4px;
      padding-right: 72px;
    }
    .address-row {
      font-family: 'Roboto Mono', monospace;
      font-size: 0.8125rem;
      color: var(--hdns-success);
      padding: 4px 8px;
      border-radius: 4px;
      word-break: break-all;
    }

    .result-error {
      display: flex;
      align-items: flex-start;
      gap: 6px;
      font-size: 0.8125rem;
      color: var(--hdns-danger);
      padding-right: 72px;
    }
    .err-icon { font-size: 16px; width: 16px; height: 16px; margin-top: 1px; flex-shrink: 0; }

    .pending-msg { font-size: 0.8125rem; color: var(--launch-text-muted); font-style: italic; }
    .drawer-footer {
      flex-shrink: 0;
      padding: 12px 24px;
      border-top: 1px solid var(--launch-border-color);
      display: flex; justify-content: flex-end;
    }
  `],
})
export class ResolveDrawerComponent implements OnChanges, OnDestroy {
  record: DnsRecord | null = null;
  @ViewChild('drawer') drawer!: DrawerComponent;

  results: Map<string, Resolution> = new Map();

  private stream: Stream<Resolution, unknown> | null = null;
  private messagesSub?: Subscription;
  private connectedSub?: Subscription;

  constructor(
    private readonly api: ApiService,
    private readonly cdr: ChangeDetectorRef,
    private readonly notify: NotifyService,
  ) {}

  ngOnChanges(): void {
    this.closeStream();
    if (this.record) {
      this.startStream();
    }
  }

  ngOnDestroy(): void {
    this.closeStream();
  }

  open(r: DnsRecord): void {
    this.record = r;
    this.startStream();
    this.drawer.open();
  }

  close(): void {
    this.closeStream();
    this.drawer.close();
  }

  startStream(): void {
    this.results = new Map();
    this.stream = this.api.streamResolveRecord();
    
    this.connectedSub = this.stream.connect$.subscribe(() => {
      this.stream.send(this.record!);
    });

    this.messagesSub = this.stream.messages$.subscribe({
      next: (msg) => {
        this.results = new Map(this.results.set(msg.server, msg));
        this.cdr.markForCheck();
      },
      error: (err) => {
        this.notify.error(err?.message ?? String(err), 'Resolve stream error');
      },
    });
  }

  closeStream(): void {
    this.messagesSub?.unsubscribe();
    this.messagesSub = undefined;
    this.connectedSub?.unsubscribe();
    this.connectedSub = undefined;
    this.stream?.close();
    this.stream = null;
  }
}
