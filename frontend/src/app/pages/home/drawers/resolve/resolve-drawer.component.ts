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
    <app-drawer #drawer [width]="45" [breakpoints]="[{ maxWidth: 768, width: 100 }]">
    <div class="drawer-header" header>
      <h3 class="drawer-title">Resolve: {{ record?.name }}.{{ record?.domain }}</h3>
    </div>

    <div class="drawer-body" content>

      @if (results.size === 0) {
        <div class="empty-msg">
          <mat-icon>travel_explore</mat-icon>
          <span>No results yet.</span>
        </div>
      }

      <div class="result-list">
        @for (r of results.values(); track r.server) {
          <div class="result-row">
            <div class="result-server">{{ r.server }}</div>
            @if (r.addresses?.length) {
              <div class="result-addresses">
                @for (a of r.addresses; track a) {
                  <span class="mono">{{ a }}</span>
                }
              </div>
            }
            @if (r.error) {
              <div class="result-error">
                <mat-icon class="err-icon">error_outline</mat-icon>{{ r.error }}
              </div>
            }
            <div class="result-meta">
              @if (r.responseTime) {
                <span>{{ r.responseTime }}ms</span>
              }
            </div>
          </div>
        }
      </div>
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
    .empty-msg { display: flex; align-items: center; gap: 8px; color: var(--launch-text-muted); font-size: 0.875rem; }
    .result-list { display: flex; flex-direction: column; gap: 12px; }
    .result-row {
      background: #16161e; border: 1px solid var(--launch-border-color);
      border-radius: 8px; padding: 12px 14px;
      display: flex; flex-direction: column; gap: 4px;
    }
    .result-server { font-size: 0.8125rem; font-weight: 600; color: var(--launch-text-primary); }
    .result-addresses { display: flex; flex-wrap: wrap; gap: 6px; }
    .mono { font-family: 'Roboto Mono', monospace; font-size: 0.75rem; color: var(--hdns-primary); background: var(--hdns-primary-tint); padding: 2px 6px; border-radius: 4px; }
    .result-error { display: flex; align-items: center; gap: 4px; font-size: 0.8125rem; color: var(--hdns-danger); }
    .err-icon { font-size: 14px; width: 14px; height: 14px; }
    .result-meta { font-size: 0.75rem; color: var(--launch-text-muted); }
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
