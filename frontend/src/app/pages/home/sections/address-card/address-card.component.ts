import { CommonModule } from '@angular/common';
import { Component, effect, inject, OnDestroy, signal } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatCardModule } from '@angular/material/card';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { DrawerComponent } from '../../../../components/drawer/drawer.component';
import { ApiService } from '../../../../global/services/api/api.service';
import { NotifyService } from '../../../../global/services/notify/notify.service';
import { AddressHistoryDrawerComponent } from '../../drawers/address-history/address-history-drawer.component';

@Component({
  selector: 'app-address-card',
  standalone: true,
  imports: [
    CommonModule,
    MatCardModule,
    MatButtonModule,
    MatIconModule,
    MatTooltipModule,
    DrawerComponent,
    AddressHistoryDrawerComponent,
  ],
  template: `
    <div class="hdns-card address-card">
      <div class="card-header">
        <div class="card-title-row">
          <mat-icon class="card-icon">dns</mat-icon>
          <h2 class="card-title">Current Address</h2>
          <span class="live-badge" *ngIf="address()">
            <span class="live-dot"></span>Live
          </span>
        </div>
        <div class="card-actions">
          <button mat-stroked-button (click)="refresh()" [disabled]="refreshing()">
            <mat-icon>refresh</mat-icon>
            Refresh now
          </button>
          <button mat-stroked-button (click)="historyDrawer.open()">
            <mat-icon>history</mat-icon>
            View history
          </button>
        </div>
      </div>

      <div class="card-body" *ngIf="address(); else noAddress">
        <div class="address-row">
          <span class="address-label">IPv4</span>
          <span class="address-value">{{ address()!.ipv4 || '—' }}</span>
        </div>
        <div class="address-row">
          <span class="address-label">IPv6</span>
          <span class="address-value">{{ address()!.ipv6 || '—' }}</span>
        </div>
        <div class="address-row" *ngIf="address()!.updatedAt">
          <span class="address-label">Last updated</span>
          <span class="address-value muted">{{ address()!.updatedAt | date:'medium' }}</span>
        </div>
      </div>

      <ng-template #noAddress>
        <div class="card-body no-data">
          <mat-icon>wifi_off</mat-icon>
          <span>Connecting…</span>
        </div>
      </ng-template>
    </div>

    <app-drawer #historyDrawer [width]="40" [breakpoints]="[{ maxWidth: 768, width: 100 }]">
      <app-address-history-drawer (close)="historyDrawer.close()" />
    </app-drawer>
  `,
  styles: [`
    .address-card {
      padding: 20px 24px;
    }
    .card-header {
      display: flex;
      align-items: flex-start;
      justify-content: space-between;
      gap: 12px;
      flex-wrap: wrap;
      margin-bottom: 16px;
    }
    .card-title-row {
      display: flex;
      align-items: center;
      gap: 8px;
    }
    .card-icon { color: var(--hdns-primary); font-size: 20px; }
    .card-title {
      margin: 0;
      font-size: 1rem;
      font-weight: 600;
      color: var(--launch-text-primary);
    }
    .live-badge {
      display: flex;
      align-items: center;
      gap: 4px;
      font-size: 0.6875rem;
      font-weight: 500;
      color: var(--hdns-success);
      background: rgba(45, 211, 111, 0.1);
      padding: 2px 8px;
      border-radius: 99px;
    }
    .live-dot {
      width: 6px; height: 6px;
      border-radius: 50%;
      background: var(--hdns-success);
    }
    .card-actions { display: flex; gap: 8px; flex-wrap: wrap; }
    .card-body { display: flex; flex-direction: column; gap: 8px; }
    .address-row {
      display: flex;
      align-items: center;
      gap: 12px;
      font-size: 0.875rem;
    }
    .address-label {
      width: 96px;
      color: var(--launch-text-muted);
      font-weight: 500;
      flex-shrink: 0;
    }
    .address-value {
      font-family: 'Roboto Mono', monospace;
      color: var(--launch-text-primary);
    }
    .address-value.muted { font-family: inherit; color: var(--launch-text-secondary); }
    .no-data {
      flex-direction: row !important;
      align-items: center;
      color: var(--launch-text-muted);
      gap: 8px;
      font-size: 0.875rem;
    }
  `],
})
export class AddressCardComponent implements OnDestroy {
  private readonly api = inject(ApiService);
  private readonly stream = this.api.streamAddress();
  private readonly notify = inject(NotifyService);

  readonly address = this.stream.latestMessage;
  readonly refreshing = signal(false);

  constructor() {
    effect(() => {
      if (this.stream.isConnected()) {
        this.stream.send({});
      }
    });
  }

  ngOnDestroy(): void {
    this.stream.close();
  }

  refresh(): void {
    this.refreshing.set(true);
    this.notify.loading();
    this.api.refreshAddress().subscribe({
      next: () => {
        this.notify.dismiss();
        this.notify.message('Address refreshed successfully.');
        this.refreshing.set(false);
      },
      error: (err) => {
        this.notify.dismiss();
        this.notify.error(err?.error?.message ?? String(err), 'Refresh failed');
        this.refreshing.set(false);
      },
    });
  }
}
