import { CommonModule } from '@angular/common';
import { Component, OnInit, signal, ViewChild } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatCheckboxModule } from '@angular/material/checkbox';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
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
    FormsModule,
    MatTableModule,
    MatButtonModule,
    MatIconModule,
    MatMenuModule,
    MatTooltipModule,
    MatCheckboxModule,
    MatFormFieldModule,
    MatInputModule,
    RecordFormDrawerComponent,
    ResolveDrawerComponent,
  ],
  template: `
    <div class="hdns-card records-card">
      <div class="card-header">
        <div class="card-title-row">
          <mat-icon class="card-icon">table_rows</mat-icon>
          <h2 class="card-title">DNS Records</h2>
          @if (records().length) {
            <span class="record-count">{{ records().length }}</span>
          }
        </div>
        <div class="card-actions">
          <mat-form-field appearance="outline" class="search-field" subscriptSizing="dynamic">
            <mat-icon matPrefix>search</mat-icon>
            <input
              matInput
              type="search"
              placeholder="Search by name or domain"
              [ngModel]="searchTerm()"
              (ngModelChange)="searchTerm.set($event); load($event)"
            />
            @if (searchTerm()) {
              <button
                mat-icon-button
                matSuffix
                type="button"
                class="clear-button"
                aria-label="Clear search"
                (click)="searchTerm.set(''); load('')"
              >
                <mat-icon>close</mat-icon>
              </button>
            }
          </mat-form-field>
          <button mat-flat-button color="primary" (click)="recordDrawer.open()">
            <mat-icon>add</mat-icon>
            Add record
          </button>
        </div>
      </div>

      @if (records().length > 0) {
        <div class="table-container">
          <table mat-table [dataSource]="records()" class="records-table">
            <ng-container matColumnDef="name">
              <th mat-header-cell *matHeaderCellDef>Name</th>
              <td mat-cell *matCellDef="let r">{{ r.name }}</td>
            </ng-container>

            <ng-container matColumnDef="domain">
              <th mat-header-cell *matHeaderCellDef>Domain</th>
              <td mat-cell *matCellDef="let r">{{ r.domain }}</td>
            </ng-container>

            <ng-container matColumnDef="address">
              <th mat-header-cell *matHeaderCellDef>Resolved IP</th>
              <td mat-cell *matCellDef="let r">
                <span class="mono">{{ r.address?.ipv4 || r.address?.ipv6 || '—' }}</span>
              </td>
            </ng-container>

            <ng-container matColumnDef="updatedAt">
              <th mat-header-cell *matHeaderCellDef>Last Updated</th>
              <td mat-cell *matCellDef="let r" class="muted">
                {{ r.updatedAt ? (r.updatedAt | date: 'dd.MM.yyyy HH:mm' : 'UTC') : '—' }}
              </td>
            </ng-container>

            <ng-container matColumnDef="ttl">
              <th mat-header-cell *matHeaderCellDef>TTL</th>
              <td mat-cell *matCellDef="let r">{{ r.ttl }}s</td>
            </ng-container>

            <ng-container matColumnDef="actions">
              <th mat-header-cell *matHeaderCellDef></th>
              <td mat-cell *matCellDef="let r" class="actions-cell">
                <div class="actions-row">
                  <button mat-button class="primary-item" (click)="refreshRecord(r)">
                    <mat-icon>refresh</mat-icon> Refresh
                  </button>
                  <button mat-button class="primary-item" (click)="resolveDrawer.open(r)">
                    <mat-icon>travel_explore</mat-icon> Resolve
                  </button>
                  <button mat-button class="danger-item" (click)="delete(r)">
                    <mat-icon>delete</mat-icon> Delete
                  </button>
                </div>
              </td>
            </ng-container>

            <tr mat-header-row *matHeaderRowDef="columns"></tr>
            <tr mat-row *matRowDef="let row; columns: columns"></tr>
          </table>
        </div>
      } @else if (records().length > 0) {
        <div class="empty-state">
          <mat-icon>search_off</mat-icon>
          <p>No records match "{{ searchTerm() }}".</p>
        </div>
      } @else {
        <div class="empty-state">
          <mat-icon>dns</mat-icon>
          <p>No records yet. Add your first DNS record to get started.</p>
        </div>
      }
    </div>

    <!-- Edit / Add record drawer -->
    <app-record-form-drawer #recordDrawer (onSave)="load()" />

    <!-- Resolve drawer -->
    <app-resolve-drawer #resolveDrawer />
  `,
  styles: [
    `
      .records-card {
        padding: 20px 24px;
      }
      .card-header {
        display: flex;
        align-items: center;
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
      .card-actions {
        display: flex;
        align-items: center;
        gap: 12px;
        flex-wrap: wrap;
      }
      .search-field {
        width: 280px;
        max-width: 100%;
        font-size: 0.875rem;
      }
      .search-field ::ng-deep .mat-mdc-text-field-wrapper {
        height: 40px;
      }
      .search-field ::ng-deep .mat-mdc-form-field-infix {
        min-height: 40px;
        padding-top: 8px;
        padding-bottom: 8px;
      }
      .search-field .clear-button {
        width: 32px;
        height: 32px;
        padding: 0;
        line-height: 32px;
      }
      .search-field .clear-button mat-icon {
        font-size: 18px;
        width: 18px;
        height: 18px;
        line-height: 18px;
      }
      .search-field mat-icon[matPrefix] {
        color: var(--launch-text-muted);
        margin-right: 6px;
      }
      .card-icon {
        color: var(--hdns-primary);
        font-size: 20px;
      }
      .card-title {
        margin: 0;
        font-size: 1rem;
        font-weight: 600;
        color: var(--launch-text-primary);
      }
      .record-count {
        background: var(--hdns-primary-tint);
        color: var(--hdns-primary);
        font-size: 0.75rem;
        font-weight: 600;
        padding: 2px 8px;
        border-radius: 99px;
      }
      .table-container {
        overflow-x: auto;
      }
      .records-table {
        width: 100%;
      }
      .mono {
        font-family: 'Roboto Mono', monospace;
        font-size: 0.8125rem;
      }
      .muted {
        color: var(--launch-text-muted);
        font-size: 0.8125rem;
      }
      .actions-cell {
        white-space: nowrap;
        padding-right: 0;
      }
      .actions-row {
        display: flex;
        flex-direction: row;
        align-items: center;
        justify-content: flex-end;
        gap: 4px;
        padding: 0 4px;
      }
      .empty-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        padding: 48px 24px;
        gap: 12px;
        color: var(--launch-text-muted);
      }
      .empty-state mat-icon {
        font-size: 40px;
        width: 40px;
        height: 40px;
      }
      .empty-state p {
        margin: 0;
        font-size: 0.875rem;
        text-align: center;
      }
      .danger-item {
        color: var(--hdns-danger) !important;
      }
      .danger-item mat-icon {
        color: var(--hdns-danger) !important;
      }
      .primary-item {
        color: var(--hdns-primary) !important;
      }
      .primary-item mat-icon {
        color: var(--hdns-primary) !important;
      }
    `,
  ],
})
export class RecordsTableComponent implements OnInit {
  readonly records = signal<DnsRecord[]>([]);
  readonly searchTerm = signal<string>('');
  columns = ['name', 'domain', 'address', 'updatedAt', 'ttl', 'actions'];

  @ViewChild('resolveDrawer') resolveDrawer!: DrawerComponent;

  constructor(
    private readonly api: ApiService,
    private readonly notify: NotifyService,
  ) {}

  ngOnInit(): void {
    this.load();
  }

  load(search?: string): void {
    this.api
      .getRecords({
        search: search,
      })
      .subscribe({
        next: (res) => this.records.set(res.records ?? []),
        error: (err) => this.notify.error(err?.error, 'Failed to load records'),
      });
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
        this.notify.error(err?.error, 'Refresh failed');
      },
    });
  }

  delete(record: DnsRecord): void {
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
      showCheckbox: true,
      checkboxLabel: 'Delete record from Hetzner',
      checkboxValue: deleteFromHetzner,
      checkboxChange: (checked) => (deleteFromHetzner = checked),
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
        this.notify.error(err?.error, 'Delete failed');
      },
    });
  }
}
