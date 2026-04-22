import { CdkDrag, CdkDragDrop, CdkDropList, moveItemInArray } from '@angular/cdk/drag-drop';
import { CommonModule } from '@angular/common';
import { Component, inject, OnInit, signal, ViewChild } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { DrawerComponent } from '../../../../components/drawer/drawer.component';
import { Configuration } from '../../../../global/model/api';
import { ApiService } from '../../../../global/services/api/api.service';
import { NotifyService } from '../../../../global/services/notify/notify.service';

const LOG_LEVELS = [
  { value: -1, label: 'Trace' },
  { value: 0, label: 'Debug' },
  { value: 1, label: 'Info' },
  { value: 2, label: 'Warning' },
  { value: 3, label: 'Error' },
];

@Component({
  selector: 'app-config-drawer',
  standalone: true,
  imports: [
    CommonModule,
    DrawerComponent,
    ReactiveFormsModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatButtonModule,
    MatIconModule,
    CdkDropList,
    CdkDrag,
  ],
  template: `
    <app-drawer #drawer (closed)="drawer.close()" [width]="45" [breakpoints]="[{ maxWidth: 768, width: 100 }, { maxWidth: 1200, width: 80 }, { maxWidth: 1600, width: 60 }]">
      <div class="drawer-header" header>
        <h3 class="drawer-title">Configuration</h3>
      </div>

      <div class="drawer-body" content>
        @if (loading()) {
          <div class="loading-msg">Loading configuration…</div>
        } @else {
          <form [formGroup]="form" class="config-form">
            <mat-form-field appearance="outline" class="full-width">
              <mat-label>Log Level</mat-label>
              <mat-select formControlName="logLevel">
                @for (l of logLevels; track l.value) {
                  <mat-option [value]="l.value">{{ l.label }}</mat-option>
                }
              </mat-select>
            </mat-form-field>

            <mat-form-field appearance="outline" class="full-width">
              <mat-label>Refresh Cron</mat-label>
              <input matInput formControlName="refreshCron" placeholder="e.g. @every 5m" />
            </mat-form-field>

            <div class="list-section">
              <div class="list-header">
                <label class="list-label">DNS Servers</label>
                <span class="list-count">{{ (form.value.dnsServers ?? []).length }}</span>
              </div>
              <div class="add-row">
                <input
                  #dnsInput
                  class="list-input"
                  placeholder="Add server address"
                  (keydown.enter)="addToArray('dnsServers', dnsInput.value); dnsInput.value = ''"
                />
                <button
                  type="button"
                  mat-icon-button
                  class="add-btn"
                  (click)="addToArray('dnsServers', dnsInput.value); dnsInput.value = ''"
                  aria-label="Add DNS server"
                >
                  <mat-icon>add</mat-icon>
                </button>
              </div>
              @if ((form.value.dnsServers ?? []).length === 0) {
                <div class="list-empty">No DNS servers configured.</div>
              } @else {
                <div
                  class="list-items"
                  cdkDropList
                  (cdkDropListDropped)="onDrop('dnsServers', $event)"
                >
                  @for (s of form.value.dnsServers; track $index; let i = $index) {
                    <div class="list-item" cdkDrag cdkDragLockAxis="y">
                      <mat-icon class="drag-handle" cdkDragHandle aria-label="Reorder">drag_indicator</mat-icon>
                      <span class="list-index">{{ i + 1 }}</span>
                      <span class="list-value">{{ s }}</span>
                      <button
                        type="button"
                        mat-icon-button
                        class="remove-btn"
                        (click)="removeFromArray('dnsServers', i)"
                        aria-label="Remove entry"
                      >
                        <mat-icon>close</mat-icon>
                      </button>
                    </div>
                  }
                </div>
              }
            </div>

            <div class="list-section">
              <div class="list-header">
                <label class="list-label">IPv4 Resolvers</label>
                <span class="list-count">{{ (form.value.ipv4Resolvers ?? []).length }}</span>
              </div>
              <div class="add-row">
                <input
                  #ipv4Input
                  class="list-input"
                  placeholder="Add IPv4 resolver"
                  (keydown.enter)="addToArray('ipv4Resolvers', ipv4Input.value); ipv4Input.value = ''"
                />
                <button
                  type="button"
                  mat-icon-button
                  class="add-btn"
                  (click)="addToArray('ipv4Resolvers', ipv4Input.value); ipv4Input.value = ''"
                  aria-label="Add IPv4 resolver"
                >
                  <mat-icon>add</mat-icon>
                </button>
              </div>
              @if ((form.value.ipv4Resolvers ?? []).length === 0) {
                <div class="list-empty">No IPv4 resolvers configured.</div>
              } @else {
                <div
                  class="list-items"
                  cdkDropList
                  (cdkDropListDropped)="onDrop('ipv4Resolvers', $event)"
                >
                  @for (r of form.value.ipv4Resolvers; track $index; let i = $index) {
                    <div class="list-item" cdkDrag cdkDragLockAxis="y">
                      <mat-icon class="drag-handle" cdkDragHandle aria-label="Reorder">drag_indicator</mat-icon>
                      <span class="list-index">{{ i + 1 }}</span>
                      <span class="list-value">{{ r }}</span>
                      <button
                        type="button"
                        mat-icon-button
                        class="remove-btn"
                        (click)="removeFromArray('ipv4Resolvers', i)"
                        aria-label="Remove entry"
                      >
                        <mat-icon>close</mat-icon>
                      </button>
                    </div>
                  }
                </div>
              }
            </div>

            <div class="list-section">
              <div class="list-header">
                <label class="list-label">IPv6 Resolvers</label>
                <span class="list-count">{{ (form.value.ipv6Resolvers ?? []).length }}</span>
              </div>
              <div class="add-row">
                <input
                  #ipv6Input
                  class="list-input"
                  placeholder="Add IPv6 resolver"
                  (keydown.enter)="addToArray('ipv6Resolvers', ipv6Input.value); ipv6Input.value = ''"
                />
                <button
                  type="button"
                  mat-icon-button
                  class="add-btn"
                  (click)="addToArray('ipv6Resolvers', ipv6Input.value); ipv6Input.value = ''"
                  aria-label="Add IPv6 resolver"
                >
                  <mat-icon>add</mat-icon>
                </button>
              </div>
              @if ((form.value.ipv6Resolvers ?? []).length === 0) {
                <div class="list-empty">No IPv6 resolvers configured.</div>
              } @else {
                <div
                  class="list-items"
                  cdkDropList
                  (cdkDropListDropped)="onDrop('ipv6Resolvers', $event)"
                >
                  @for (r of form.value.ipv6Resolvers; track $index; let i = $index) {
                    <div class="list-item" cdkDrag cdkDragLockAxis="y">
                      <mat-icon class="drag-handle" cdkDragHandle aria-label="Reorder">drag_indicator</mat-icon>
                      <span class="list-index">{{ i + 1 }}</span>
                      <span class="list-value">{{ r }}</span>
                      <button
                        type="button"
                        mat-icon-button
                        class="remove-btn"
                        (click)="removeFromArray('ipv6Resolvers', i)"
                        aria-label="Remove entry"
                      >
                        <mat-icon>close</mat-icon>
                      </button>
                    </div>
                  }
                </div>
              }
            </div>
          </form>
        }
      </div>

      <div class="drawer-footer" footer>
        <button mat-stroked-button (click)="drawer.close()">Cancel</button>
        <button
          mat-flat-button
          color="primary"
          [disabled]="form.invalid || saving()"
          (click)="save()"
        >
          {{ saving() ? 'Saving…' : 'Save' }}
        </button>
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
      .config-form {
        display: flex;
        flex-direction: column;
        gap: 4px;
      }
      .full-width {
        width: 100%;
      }
      .field-group {
        margin-bottom: 12px;
      }
      .field-label {
        font-size: 0.8125rem;
        font-weight: 500;
        color: var(--launch-text-secondary);
        display: block;
        margin-bottom: 6px;
      }
      .list-section {
        background: #16161e;
        border: 1px solid var(--launch-border-color);
        border-radius: 8px;
        padding: 12px 12px 10px;
        margin-bottom: 12px;
      }
      .list-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-bottom: 10px;
      }
      .list-label {
        font-size: 0.8125rem;
        font-weight: 600;
        color: var(--launch-text-primary);
        text-transform: uppercase;
        letter-spacing: 0.04em;
      }
      .list-count {
        font-size: 0.6875rem;
        font-weight: 600;
        color: var(--hdns-primary);
        background: var(--hdns-primary-tint);
        padding: 2px 8px;
        border-radius: 999px;
        font-variant-numeric: tabular-nums;
        min-width: 20px;
        text-align: center;
      }
      .add-row {
        display: flex;
        align-items: center;
        gap: 6px;
        margin-bottom: 10px;
      }
      .list-input {
        flex: 1;
        min-width: 0;
        padding: 8px 12px;
        border: 1px solid var(--launch-border-color);
        border-radius: 4px;
        font-size: 0.875rem;
        font-family: inherit;
        background: #0f0f14;
        color: var(--launch-text-primary);
        outline: none;
        transition: border-color 0.15s ease;
      }
      .list-input:focus {
        border-color: var(--hdns-primary);
      }
      .add-btn {
        flex-shrink: 0;
      }
      .list-empty {
        font-size: 0.8125rem;
        color: var(--launch-text-muted);
        font-style: italic;
        padding: 8px 4px;
        text-align: center;
      }
      .list-items {
        display: flex;
        flex-direction: column;
        gap: 4px;
      }
      .list-item {
        display: flex;
        align-items: center;
        gap: 10px;
        padding: 6px 8px 6px 10px;
        background: rgba(255, 255, 255, 0.02);
        border: 1px solid var(--launch-border-color);
        border-radius: 6px;
        transition: border-color 0.15s ease, background 0.15s ease;
      }
      .list-item:hover {
        border-color: color-mix(in srgb, var(--hdns-primary) 40%, var(--launch-border-color));
        background: rgba(255, 255, 255, 0.04);
      }
      .list-index {
        font-size: 0.6875rem;
        font-weight: 600;
        color: var(--launch-text-muted);
        font-variant-numeric: tabular-nums;
        min-width: 18px;
        text-align: right;
      }
      .list-value {
        flex: 1;
        min-width: 0;
        font-family: 'Roboto Mono', monospace;
        font-size: 0.8125rem;
        color: var(--launch-text-primary);
        word-break: break-all;
      }
      .remove-btn {
        flex-shrink: 0;
        width: 32px;
        height: 32px;
        padding: 0;
        line-height: 1;
        display: inline-flex;
        align-items: center;
        justify-content: center;
      }
      .remove-btn mat-icon {
        font-size: 18px;
        width: 18px;
        height: 18px;
        line-height: 18px;
        margin: 0;
      }
      .drag-handle {
        flex-shrink: 0;
        cursor: grab;
        color: var(--launch-text-muted);
        font-size: 18px;
        width: 18px;
        height: 18px;
        user-select: none;
        touch-action: none;
      }
      .drag-handle:active {
        cursor: grabbing;
      }
      .list-item.cdk-drag-preview {
        box-shadow: 0 6px 14px rgba(0, 0, 0, 0.45);
        border-color: var(--hdns-primary);
        background: #1a1a22;
      }
      .list-item.cdk-drag-placeholder {
        opacity: 0.3;
      }
      .list-items.cdk-drop-list-dragging .list-item:not(.cdk-drag-placeholder) {
        transition: transform 200ms cubic-bezier(0, 0, 0.2, 1);
      }
      .drawer-footer {
        flex-shrink: 0;
        padding: 12px 24px;
        border-top: 1px solid var(--launch-border-color);
        display: flex;
        justify-content: flex-end;
        gap: 8px;
      }
    `,
  ],
})
export class ConfigDrawerComponent implements OnInit {

  @ViewChild('drawer') drawer!: DrawerComponent;

  loading = signal(true);
  saving = signal(false);
  logLevels = LOG_LEVELS;

  private readonly fb = inject(FormBuilder);
  private readonly api = inject(ApiService);
  private readonly notify = inject(NotifyService);

  form = this.fb.group({
    logLevel: [0],
    webPort: [443, [Validators.required, Validators.min(1)]],
    certificatePath: [''],
    keyPath: [''],
    refreshCron: [''],
    dnsServers: [[] as string[]],
    ipv4Resolvers: [[] as string[]],
    ipv6Resolvers: [[] as string[]],
    database: [{ value: '', disabled: true }],
  });

  constructor() {}

  ngOnInit(): void {}

  addToArray(field: 'dnsServers' | 'ipv4Resolvers' | 'ipv6Resolvers', value: string): void {
    const trimmed = value.trim();
    if (!trimmed) return;
    const current: string[] = this.form.get(field)!.value ?? [];
    this.form.get(field)!.setValue([...current, trimmed]);
  }

  removeFromArray(field: 'dnsServers' | 'ipv4Resolvers' | 'ipv6Resolvers', index: number): void {
    const current: string[] = this.form.get(field)!.value ?? [];
    this.form.get(field)!.setValue(current.filter((_, i) => i !== index));
  }

  onDrop(
    field: 'dnsServers' | 'ipv4Resolvers' | 'ipv6Resolvers',
    event: CdkDragDrop<string[]>,
  ): void {
    if (event.previousIndex === event.currentIndex) return;
    const current: string[] = [...((this.form.get(field)!.value as string[]) ?? [])];
    moveItemInArray(current, event.previousIndex, event.currentIndex);
    this.form.get(field)!.setValue(current);
  }

  open() {
    this.loading.set(true);
    this.drawer.open();
    this.api.getConfig().subscribe({
      next: (cfg) => {
        this.loading.set(false);
        this.form.patchValue({
          logLevel: cfg.logLevel ?? 0,
          refreshCron: cfg.refreshCron,
          dnsServers: cfg.dnsServers ?? [],
          ipv4Resolvers: cfg.ipv4Resolvers ?? [],
          ipv6Resolvers: cfg.ipv6Resolvers ?? [],
        });
      },
      error: (err) => {
        this.loading.set(false);
        this.notify.error(err?.error, 'Failed to load config');
      },
    });
  }

  save(): void {
    const v = this.form.getRawValue();
    const payload: Configuration = {
      logLevel: v.logLevel ?? 0,
      refreshCron: v.refreshCron ?? '',
      dnsServers: v.dnsServers ?? [],
      ipv4Resolvers: v.ipv4Resolvers ?? [],
      ipv6Resolvers: v.ipv6Resolvers ?? [],
    };
    this.saving.set(true);
    this.api.updateConfig(payload).subscribe({
      next: () => {
        this.saving.set(false);
        this.notify.message('Configuration saved.');
        this.drawer.close();
      },
      error: (err) => {
        this.saving.set(false);
        this.notify.error(err?.error, 'Save failed');
      },
    });
  }
}
