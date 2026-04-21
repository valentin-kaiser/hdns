import { CommonModule } from '@angular/common';
import { Component, EventEmitter, inject, OnInit, Output } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatChipsModule } from '@angular/material/chips';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { Configuration } from '../../../../global/model/api';
import { ApiService } from '../../../../global/services/api/api.service';
import { NotifyService } from '../../../../global/services/notify/notify.service';

const LOG_LEVELS = [
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
    ReactiveFormsModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatButtonModule,
    MatIconModule,
    MatChipsModule,
  ],
  template: `
    <div class="drawer-header" header>
      <h3 class="drawer-title">Configuration</h3>
    </div>

    <div class="drawer-body" content>
      <div *ngIf="loading" class="loading-msg">Loading configuration…</div>

      <form *ngIf="!loading" [formGroup]="form" class="config-form">
        <mat-form-field appearance="outline" class="full-width">
          <mat-label>Log Level</mat-label>
          <mat-select formControlName="logLevel">
            <mat-option *ngFor="let l of logLevels" [value]="l.value">{{ l.label }}</mat-option>
          </mat-select>
        </mat-form-field>

        <mat-form-field appearance="outline" class="full-width">
          <mat-label>Web Port</mat-label>
          <input matInput type="number" formControlName="webPort" />
        </mat-form-field>

        <mat-form-field appearance="outline" class="full-width">
          <mat-label>Certificate Path</mat-label>
          <input matInput formControlName="certificatePath" />
          <mat-icon matSuffix>lock</mat-icon>
        </mat-form-field>

        <mat-form-field appearance="outline" class="full-width">
          <mat-label>Key Path</mat-label>
          <input matInput formControlName="keyPath" />
          <mat-icon matSuffix>vpn_key</mat-icon>
        </mat-form-field>

        <mat-form-field appearance="outline" class="full-width">
          <mat-label>Refresh Cron</mat-label>
          <input matInput formControlName="refreshCron" placeholder="e.g. @every 5m" />
        </mat-form-field>

        <div class="field-group">
          <label class="field-label">DNS Servers</label>
          <div class="chip-input-row">
            <input
              #dnsInput
              class="chip-input"
              placeholder="Add server and press Enter"
              (keydown.enter)="addToArray('dnsServers', dnsInput.value); dnsInput.value = ''"
            />
          </div>
          <mat-chip-set>
            <mat-chip
              *ngFor="let s of form.value.dnsServers; let i = index"
              [removable]="true"
              (removed)="removeFromArray('dnsServers', i)"
            >
              {{ s }}<mat-icon matChipRemove>cancel</mat-icon>
            </mat-chip>
          </mat-chip-set>
        </div>

        <div class="field-group">
          <label class="field-label">IPv4 Resolvers</label>
          <div class="chip-input-row">
            <input
              #ipv4Input
              class="chip-input"
              placeholder="Add resolver and press Enter"
              (keydown.enter)="addToArray('ipv4Resolvers', ipv4Input.value); ipv4Input.value = ''"
            />
          </div>
          <mat-chip-set>
            <mat-chip
              *ngFor="let r of form.value.ipv4Resolvers; let i = index"
              [removable]="true"
              (removed)="removeFromArray('ipv4Resolvers', i)"
            >
              {{ r }}<mat-icon matChipRemove>cancel</mat-icon>
            </mat-chip>
          </mat-chip-set>
        </div>

        <div class="field-group">
          <label class="field-label">IPv6 Resolvers</label>
          <div class="chip-input-row">
            <input
              #ipv6Input
              class="chip-input"
              placeholder="Add resolver and press Enter"
              (keydown.enter)="addToArray('ipv6Resolvers', ipv6Input.value); ipv6Input.value = ''"
            />
          </div>
          <mat-chip-set>
            <mat-chip
              *ngFor="let r of form.value.ipv6Resolvers; let i = index"
              [removable]="true"
              (removed)="removeFromArray('ipv6Resolvers', i)"
            >
              {{ r }}<mat-icon matChipRemove>cancel</mat-icon>
            </mat-chip>
          </mat-chip-set>
        </div>

        <mat-form-field appearance="outline" class="full-width">
          <mat-label>Database (read-only)</mat-label>
          <input matInput formControlName="database" [readonly]="true" />
        </mat-form-field>
      </form>
    </div>

    <div class="drawer-footer" footer>
      <button mat-stroked-button (click)="cancelled.emit()">Cancel</button>
      <button mat-flat-button color="primary" [disabled]="form.invalid || saving" (click)="save()">
        {{ saving ? 'Saving…' : 'Save' }}
      </button>
    </div>
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
      .chip-input-row {
        margin-bottom: 6px;
      }
      .chip-input {
        width: 100%;
        padding: 8px 12px;
        border: 1px solid var(--launch-border-color);
        border-radius: 4px;
        font-size: 0.875rem;
        font-family: inherit;
        background: #16161e;
        color: var(--launch-text-primary);
        outline: none;
      }
      .chip-input:focus {
        border-color: var(--hdns-primary);
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
  @Output() saved = new EventEmitter<void>();
  @Output() cancelled = new EventEmitter<void>();

  loading = true;
  saving = false;
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

  ngOnInit(): void {
    this.api.getConfig().subscribe({
      next: (cfg) => {
        this.form.patchValue({
          logLevel: cfg.logLevel,
          webPort: cfg.webPort,
          certificatePath: cfg.certificatePath,
          keyPath: cfg.keyPath,
          refreshCron: cfg.refreshCron,
          dnsServers: cfg.dnsServers ?? [],
          ipv4Resolvers: cfg.ipv4Resolvers ?? [],
          ipv6Resolvers: cfg.ipv6Resolvers ?? [],
          database: cfg.database,
        });
        this.loading = false;
      },
      error: (err) => {
        this.loading = false;
        this.notify.error(err?.error?.message ?? String(err), 'Failed to load config');
      },
    });
  }

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

  save(): void {
    const v = this.form.getRawValue();
    const payload: Configuration = {
      logLevel: v.logLevel ?? 0,
      webPort: v.webPort ?? 443,
      certificatePath: v.certificatePath ?? '',
      keyPath: v.keyPath ?? '',
      refreshCron: v.refreshCron ?? '',
      dnsServers: v.dnsServers ?? [],
      ipv4Resolvers: v.ipv4Resolvers ?? [],
      ipv6Resolvers: v.ipv6Resolvers ?? [],
      database: v.database ?? '',
    };
    this.saving = true;
    this.api.updateConfig(payload).subscribe({
      next: () => {
        this.saving = false;
        this.notify.message('Configuration saved.');
        this.saved.emit();
      },
      error: (err) => {
        this.saving = false;
        this.notify.error(err?.error?.message ?? String(err), 'Save failed');
      },
    });
  }
}
