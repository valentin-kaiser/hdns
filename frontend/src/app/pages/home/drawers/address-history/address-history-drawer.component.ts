import { CommonModule } from '@angular/common';
import { Component, EventEmitter, OnInit, Output } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { Address } from '../../../../global/model/api';
import { ApiService } from '../../../../global/services/api/api.service';
import { NotifyService } from '../../../../global/services/notify/notify.service';

@Component({
  selector: 'app-address-history-drawer',
  standalone: true,
  imports: [CommonModule, MatIconModule, MatButtonModule],
  template: `
    <div class="drawer-header" header>
      <h3 class="drawer-title">Address History</h3>
    </div>

    <div class="drawer-body" content>
      <div *ngIf="loading" class="loading-msg">Loading…</div>

      <div class="history-list" *ngIf="!loading">
        <div class="history-row" *ngFor="let a of addresses" [class.current]="a.current">
          <div class="history-meta">
            <span class="history-date">{{ a.updatedAt | date:'medium' }}</span>
            <span class="current-badge" *ngIf="a.current">Current</span>
          </div>
          <div class="history-ips">
            <span class="ip-row" *ngIf="a.ipv4"><span class="label">IPv4</span><span class="mono">{{ a.ipv4 }}</span></span>
            <span class="ip-row" *ngIf="a.ipv6"><span class="label">IPv6</span><span class="mono">{{ a.ipv6 }}</span></span>
          </div>
        </div>

        <div *ngIf="addresses.length === 0" class="empty-msg">
          <mat-icon>history</mat-icon>
          <span>No history available.</span>
        </div>
      </div>
    </div>

    <div class="drawer-footer" footer>
      <button mat-stroked-button (click)="close.emit()">Close</button>
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
    }
    .loading-msg { color: var(--launch-text-muted); font-size: 0.875rem; }
    .history-list { display: flex; flex-direction: column; gap: 10px; }
    .history-row {
      background: #16161e; border: 1px solid var(--launch-border-color);
      border-radius: 8px; padding: 12px 14px;
      display: flex; flex-direction: column; gap: 6px;
    }
    .history-row.current {
      border-color: var(--hdns-primary);
      background: rgba(255, 102, 0, 0.04);
    }
    .history-meta { display: flex; align-items: center; gap: 8px; }
    .history-date { font-size: 0.8125rem; color: var(--launch-text-secondary); }
    .current-badge {
      font-size: 0.6875rem; font-weight: 600;
      background: var(--hdns-primary); color: #fff;
      padding: 1px 7px; border-radius: 99px;
    }
    .history-ips { display: flex; flex-direction: column; gap: 4px; }
    .ip-row { display: flex; align-items: center; gap: 8px; font-size: 0.8125rem; }
    .label { width: 36px; color: var(--launch-text-muted); font-weight: 500; }
    .mono { font-family: 'Roboto Mono', monospace; color: var(--launch-text-primary); }
    .empty-msg { display: flex; align-items: center; gap: 8px; color: var(--launch-text-muted); font-size: 0.875rem; }
    .drawer-footer {
      flex-shrink: 0;
      padding: 12px 24px;
      border-top: 1px solid var(--launch-border-color);
      display: flex; justify-content: flex-end;
    }
  `],
})
export class AddressHistoryDrawerComponent implements OnInit {
  @Output() close = new EventEmitter<void>();

  addresses: Address[] = [];
  loading = true;

  constructor(
    private readonly api: ApiService,
    private readonly notify: NotifyService,
  ) {}

  ngOnInit(): void {
    this.api.getAddressHistory().subscribe({
      next: (res) => {
        this.addresses = res.addresses ?? [];
        this.loading = false;
      },
      error: (err) => {
        this.loading = false;
        this.notify.error(err?.error?.message ?? String(err), 'Failed to load history');
      },
    });
  }
}
