import { CommonModule } from '@angular/common';
import { Component, OnInit, ViewChild } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatIconModule } from '@angular/material/icon';
import { MatMenuModule } from '@angular/material/menu';
import { MatTableModule } from '@angular/material/table';
import { MatTooltipModule } from '@angular/material/tooltip';
import { DrawerComponent } from '../../../../components/drawer/drawer.component';
import { Record as DnsRecord } from '../../../../global/model/api';
import { ApiService } from '../../../../global/services/api/api.service';
import { NotifyService } from '../../../../global/services/notify/notify.service';
import { RecordFormDrawerComponent } from '../../drawers/record-form/record-form-drawer.component';
import { ResolveDrawerComponent } from '../../drawers/resolve/resolve-drawer.component';

@Component({
  selector: 'app-records-table',
  standalone: true,
  imports: [
    CommonModule,
    MatTableModule,
    MatButtonModule,
    MatIconModule,
    MatMenuModule,
    MatTooltipModule,
    MatCheckboxModule,
    DrawerComponent,
    RecordFormDrawerComponent,
    ResolveDrawerComponent,
  ],
  template: `
    <div class="hdns-card records-card">
      <div class="card-header">
        <div class="card-title-row">
          <mat-icon class="card-icon">table_rows</mat-icon>
          <h2 class="card-title">DNS Records</h2>
          <span class="record-count" *ngIf="records.length">{{ records.length }}</span>
        </div>
        <button mat-flat-button color="primary" (click)="openAdd()">
          <mat-icon>add</mat-icon>
          Add record
        </button>
      </div>

      <div class="table-container" *ngIf="records.length > 0; else empty">
        <table mat-table [dataSource]="records" class="records-table">
          <ng-container matColumnDef="domain">
            <th mat-header-cell *matHeaderCellDef>Domain</th>
            <td mat-cell *matCellDef="let r">{{ r.domain }}</td>
          </ng-container>

          <ng-container matColumnDef="name">
            <th mat-header-cell *matHeaderCellDef>Name</th>
            <td mat-cell *matCellDef="let r">{{ r.name }}</td>
          </ng-container>

          <ng-container matColumnDef="ttl">
            <th mat-header-cell *matHeaderCellDef>TTL</th>
            <td mat-cell *matCellDef="let r">{{ r.ttl }}s</td>
          </ng-container>

          <ng-container matColumnDef="address">
            <th mat-header-cell *matHeaderCellDef>Resolved IP</th>
            <td mat-cell *matCellDef="let r">
              <span class="mono">{{ r.address?.ipv4 || r.address?.ipv6 || '—' }}</span>
            </td>
          </ng-container>

          <ng-container matColumnDef="lastRefresh">
            <th mat-header-cell *matHeaderCellDef>Last Refresh</th>
            <td mat-cell *matCellDef="let r" class="muted">
              {{ r.lastRefresh ? (r.lastRefresh | date:'short') : '—' }}
            </td>
          </ng-container>

          <ng-container matColumnDef="actions">
            <th mat-header-cell *matHeaderCellDef></th>
            <td mat-cell *matCellDef="let r" class="actions-cell">
              <button mat-icon-button [matMenuTriggerFor]="rowMenu" [matMenuTriggerData]="{ record: r }"
                      matTooltip="Actions" aria-label="Row actions">
                <mat-icon>more_vert</mat-icon>
              </button>
            </td>
          </ng-container>

          <tr mat-header-row *matHeaderRowDef="columns"></tr>
          <tr mat-row *matRowDef="let row; columns: columns;"></tr>
        </table>
      </div>

      <ng-template #empty>
        <div class="empty-state">
          <mat-icon>dns</mat-icon>
          <p>No records yet. Add your first DNS record to get started.</p>
        </div>
      </ng-template>
    </div>

    <!-- Row action menu -->
    <mat-menu #rowMenu="matMenu">
      <ng-template matMenuContent let-record="record">
        <button mat-menu-item (click)="openEdit(record)">
          <mat-icon>edit</mat-icon> Edit
        </button>
        <button mat-menu-item (click)="refreshRecord(record)">
          <mat-icon>refresh</mat-icon> Refresh
        </button>
        <button mat-menu-item (click)="openResolve(record)">
          <mat-icon>travel_explore</mat-icon> Resolve
        </button>
        <button mat-menu-item class="danger-item" (click)="confirmDelete(record)">
          <mat-icon color="warn">delete</mat-icon> Delete
        </button>
      </ng-template>
    </mat-menu>

    <!-- Edit / Add record drawer -->
    <app-drawer #recordDrawer [width]="45" [breakpoints]="[{ maxWidth: 768, width: 100 }]">
      <app-record-form-drawer header content footer
        [record]="editRecord"
        (saved)="onRecordSaved()"
        (cancelled)="recordDrawer.close()" />
    </app-drawer>

    <!-- Resolve drawer -->
    <app-drawer #resolveDrawer [width]="45" [breakpoints]="[{ maxWidth: 768, width: 100 }]">
      <app-resolve-drawer header content footer
        [record]="resolveRecord"
        (close)="resolveDrawer.close()" />
    </app-drawer>
  `,
  styles: [`
    .records-card { padding: 20px 24px; }
    .card-header {
      display: flex; align-items: center; justify-content: space-between;
      gap: 12px; flex-wrap: wrap; margin-bottom: 16px;
    }
    .card-title-row { display: flex; align-items: center; gap: 8px; }
    .card-icon { color: var(--hdns-primary); font-size: 20px; }
    .card-title { margin: 0; font-size: 1rem; font-weight: 600; color: var(--launch-text-primary); }
    .record-count {
      background: var(--hdns-primary-tint);
      color: var(--hdns-primary);
      font-size: 0.75rem;
      font-weight: 600;
      padding: 2px 8px;
      border-radius: 99px;
    }
    .table-container { overflow-x: auto; }
    .records-table { width: 100%; }
    .mono { font-family: 'Roboto Mono', monospace; font-size: 0.8125rem; }
    .muted { color: var(--launch-text-muted); font-size: 0.8125rem; }
    .actions-cell { width: 48px; padding-right: 0; }
    .empty-state {
      display: flex; flex-direction: column; align-items: center;
      padding: 48px 24px; gap: 12px; color: var(--launch-text-muted);
    }
    .empty-state mat-icon { font-size: 40px; width: 40px; height: 40px; }
    .empty-state p { margin: 0; font-size: 0.875rem; text-align: center; }
    .danger-item { color: var(--hdns-danger) !important; }
    .danger-item mat-icon { color: var(--hdns-danger) !important; }
  `],
})
export class RecordsTableComponent implements OnInit {
  records: DnsRecord[] = [];
  columns = ['domain', 'name', 'ttl', 'address', 'lastRefresh', 'actions'];
  editRecord: DnsRecord | null = null;
  resolveRecord: DnsRecord | null = null;

  @ViewChild('recordDrawer') recordDrawer!: DrawerComponent;
  @ViewChild('resolveDrawer') resolveDrawer!: DrawerComponent;

  constructor(
    private readonly api: ApiService,
    private readonly notify: NotifyService,
  ) {}

  ngOnInit(): void {
    this.load();
  }

  load(): void {
    this.api.getRecords().subscribe({
      next: (res) => (this.records = res.records ?? []),
      error: (err) => this.notify.error(err?.error?.message ?? String(err), 'Failed to load records'),
    });
  }

  openAdd(): void {
    this.editRecord = null;
    this.recordDrawer.open();
  }

  openEdit(record: DnsRecord): void {
    this.editRecord = record;
    this.recordDrawer.open();
  }

  openResolve(record: DnsRecord): void {
    this.resolveRecord = record;
    this.resolveDrawer.open();
  }

  onRecordSaved(): void {
    this.recordDrawer.close();
    this.load();
  }

  refreshRecord(record: DnsRecord): void {
    this.notify.loading();
    this.api.refreshRecord(record).subscribe({
      next: () => {
        this.notify.dismiss();
        this.notify.message('Record refreshed.');
        this.load();
      },
      error: (err) => {
        this.notify.dismiss();
        this.notify.error(err?.error?.message ?? String(err), 'Refresh failed');
      },
    });
  }

  confirmDelete(record: DnsRecord): void {
    let deleteFromHetzner = false;
    this.notify.warning({
      title: 'Delete record',
      message: `Delete "${record.name}.${record.domain}"? This cannot be undone.`,
      buttons: [
        { text: 'Cancel', color: 'accent' },
        {
          text: 'Delete',
          color: 'warn',
          handler: () => this.deleteRecord(record, deleteFromHetzner),
        },
      ],
    });
  }

  private deleteRecord(record: DnsRecord, deleteFromHetzner: boolean): void {
    this.notify.loading();
    this.api.deleteRecord(record, deleteFromHetzner).subscribe({
      next: () => {
        this.notify.dismiss();
        this.notify.message('Record deleted.');
        this.load();
      },
      error: (err) => {
        this.notify.dismiss();
        this.notify.error(err?.error?.message ?? String(err), 'Delete failed');
      },
    });
  }
}
