import { CommonModule } from '@angular/common';
import { Component, OnInit, signal, ViewChild } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { DrawerComponent } from '../../../../components/drawer/drawer.component';
import { Address } from '../../../../global/model/api';
import { ApiService } from '../../../../global/services/api/api.service';
import { NotifyService } from '../../../../global/services/notify/notify.service';

@Component({
  selector: 'app-address-history-drawer',
  standalone: true,
  imports: [CommonModule, MatIconModule, MatButtonModule, DrawerComponent],
  template: `
    <app-drawer #drawer (closed)="drawer.close()" [width]="40" [breakpoints]="[{ maxWidth: 768, width: 100 }, { maxWidth: 1200, width: 80 }, { maxWidth: 1600, width: 60 }]">
      <div class="drawer-header" header>
        <h3 class="drawer-title">Address History</h3>
      </div>

      <div class="drawer-body" content>
        @if (loading()) {
          <div class="loading-msg">Loading…</div>
        }

        @if (!loading()) {
          <div class="history-list">
            @for (a of addresses; track a.updatedAt) {
              <div class="history-row" [class.current]="a.current">
                <div class="history-meta">
                  <span class="history-date">{{ a.updatedAt | date: 'dd.MM.yyyy HH:mm' : 'UTC' }}</span>
                  @if (a.current) {
                    <span class="current-badge">Current</span>
                  }
                </div>
                <div class="history-ips">
                  @if (a.ipv4) {
                    <span class="ip-row"
                      ><span class="label">IPv4</span><span class="mono">{{ a.ipv4 }}</span></span
                    >
                  }
                  @if (a.ipv6) {
                    <span class="ip-row"
                      ><span class="label">IPv6</span><span class="mono">{{ a.ipv6 }}</span></span
                    >
                  }
                </div>
              </div>
            }

            @if (addresses.length === 0) {
              <div class="empty-msg">
                <mat-icon>history</mat-icon>
                <span>No history available.</span>
              </div>
            }
          </div>
        }
      </div>

      <div class="drawer-footer" footer>
        <button mat-stroked-button (click)="drawer.close()">Close</button>
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
      .history-list {
        display: flex;
        flex-direction: column;
        gap: 10px;
      }
      .history-row {
        background: #16161e;
        border: 1px solid var(--launch-border-color);
        border-radius: 8px;
        padding: 12px 14px;
        display: flex;
        flex-direction: column;
        gap: 6px;
      }
      .history-row.current {
        border-color: var(--hdns-primary);
        background: rgba(255, 102, 0, 0.04);
      }
      .history-meta {
        display: flex;
        align-items: center;
        gap: 8px;
      }
      .history-date {
        font-size: 0.8125rem;
        color: var(--launch-text-secondary);
      }
      .current-badge {
        font-size: 0.6875rem;
        font-weight: 600;
        background: var(--hdns-primary);
        color: #fff;
        padding: 1px 7px;
        border-radius: 99px;
      }
      .history-ips {
        display: flex;
        flex-direction: column;
        gap: 4px;
      }
      .ip-row {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 0.8125rem;
      }
      .label {
        width: 36px;
        color: var(--launch-text-muted);
        font-weight: 500;
      }
      .mono {
        font-family: 'Roboto Mono', monospace;
        color: var(--launch-text-primary);
      }
      .empty-msg {
        display: flex;
        align-items: center;
        gap: 8px;
        color: var(--launch-text-muted);
        font-size: 0.875rem;
      }
      .drawer-footer {
        flex-shrink: 0;
        padding: 12px 24px;
        border-top: 1px solid var(--launch-border-color);
        display: flex;
        justify-content: flex-end;
      }
    `,
  ],
})
export class AddressHistoryDrawerComponent implements OnInit {
  @ViewChild('drawer') drawer!: DrawerComponent;

  addresses: Address[] = [];
  loading = signal(true);

  constructor(
    private readonly api: ApiService,
    private readonly notify: NotifyService,
  ) {}

  ngOnInit(): void {}

  open() {
    this.drawer.open();
    this.api.getAddressHistory().subscribe({
      next: (res) => {
        this.addresses = res.addresses ?? [];
        this.loading.set(false);
      },
      error: (err) => {
        this.loading.set(false);
        this.notify.error(err?.error, 'Failed to load history');
      },
    });
  }
}
