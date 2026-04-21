import { Component, EventEmitter, Input, OnChanges, OnDestroy, Output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { ApiService, Stream } from '../../../../global/services/api/api.service';
import { Record as DnsRecord, ResolutionResult } from '../../../../global/model/api';

@Component({
  selector: 'app-resolve-drawer',
  standalone: true,
  imports: [CommonModule, MatIconModule, MatButtonModule, MatProgressSpinnerModule],
  template: `
    <div class="drawer-header" header>
      <h3 class="drawer-title">Resolve: {{ record?.name }}.{{ record?.domain }}</h3>
    </div>

    <div class="drawer-body" content>
      <div *ngIf="connecting" class="spinner-row">
        <mat-spinner diameter="24" /><span>Connecting…</span>
      </div>

      <div *ngIf="results.length === 0 && !connecting" class="empty-msg">
        <mat-icon>travel_explore</mat-icon>
        <span>No results yet.</span>
      </div>

      <div class="result-list">
        <div class="result-row" *ngFor="let r of results">
          <div class="result-server">{{ r.server }}</div>
          <div class="result-addresses" *ngIf="r.addresses?.length">
            <span class="mono" *ngFor="let a of r.addresses">{{ a }}</span>
          </div>
          <div class="result-error" *ngIf="r.error">
            <mat-icon class="err-icon">error_outline</mat-icon>{{ r.error }}
          </div>
          <div class="result-meta">
            <span *ngIf="r.responseTime">{{ r.responseTime }}ms</span>
          </div>
        </div>
      </div>
    </div>

    <div class="drawer-footer" footer>
      <button mat-stroked-button (click)="closeStream(); close.emit()">Close</button>
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
  @Input() record: DnsRecord | null = null;
  @Output() close = new EventEmitter<void>();

  results: ResolutionResult['resolutions'] = [];
  connecting = false;

  private stream: Stream<ResolutionResult, unknown> | null = null;
  private _clearInterval?: () => void;

  constructor(private readonly api: ApiService) {}

  ngOnChanges(): void {
    this.closeStream();
    if (this.record) {
      this.startStream();
    }
  }

  ngOnDestroy(): void {
    this.closeStream();
  }

  startStream(): void {
    this.results = [];
    this.connecting = true;
    this.stream = this.api.streamResolveRecord(this.record!);
    const interval = setInterval(() => {
      if (this.stream!.isConnected()) this.connecting = false;
      const msgs = this.stream!.messages();
      if (msgs.length > 0) {
        const allResolutions = msgs.flatMap((m: any) => m.resolutions ?? []);
        this.results = allResolutions;
      }
    }, 300);
    this._clearInterval = () => clearInterval(interval);
  }

  closeStream(): void {
    this.stream?.close();
    this.stream = null;
    this._clearInterval?.();
    this._clearInterval = undefined;
  }
}
