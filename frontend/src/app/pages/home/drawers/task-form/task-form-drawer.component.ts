import { CommonModule } from '@angular/common';
import { Component, ElementRef, EventEmitter, inject, Output, QueryList, ViewChild, ViewChildren } from '@angular/core';
import { AbstractControl, FormArray, FormBuilder, FormGroup, ReactiveFormsModule, ValidationErrors, Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatDividerModule } from '@angular/material/divider';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatSlideToggleModule } from '@angular/material/slide-toggle';
import { DrawerComponent } from '../../../../components/drawer/drawer.component';
import { Task } from '../../../../global/model/api';
import { ApiService } from '../../../../global/services/api/api.service';
import { NotifyService } from '../../../../global/services/notify/notify.service';

const HTTP_METHODS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'];
const CERTIFICATE_FORMATS = ['pem', 'pkcs12'] as const;
type HeaderRow = { key: string; value: string };

@Component({
  selector: 'app-task-form-drawer',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatSlideToggleModule,
    MatButtonModule,
    MatDividerModule,
    MatIconModule,
    DrawerComponent,
  ],
  template: `
    <app-drawer #drawer (closed)="drawer.close()" [width]="45" [breakpoints]="[{ maxWidth: 768, width: 100 }, { maxWidth: 1200, width: 80 }, { maxWidth: 1600, width: 60 }]">
      <div class="drawer-header">
        <h3 class="drawer-title">{{ task ? 'Edit Task' : 'Add Task' }}</h3>
      </div>

      <div class="drawer-body">
        <form [formGroup]="form" class="task-form">
          <mat-form-field appearance="outline" class="full-width">
            <mat-label>Name</mat-label>
            <input matInput formControlName="name" placeholder="e.g. Restart reverse proxy" />
          </mat-form-field>

          <mat-form-field appearance="outline" class="full-width">
            <mat-label>Run on</mat-label>
            <mat-select formControlName="triggerOn">
              <mat-option [value]="1">IP update</mat-option>
              <mat-option [value]="2">Certificate renewal</mat-option>
              <mat-option [value]="3">IP update &amp; Certificate renewal</mat-option>
            </mat-select>
          </mat-form-field>

          <div class="row">
            <mat-form-field appearance="outline" class="method-field">
              <mat-label>Method</mat-label>
              <mat-select formControlName="method">
                @for (m of methods; track m) {
                  <mat-option [value]="m">{{ m }}</mat-option>
                }
              </mat-select>
            </mat-form-field>
            <mat-form-field appearance="outline" class="url-field">
              <mat-label>Webhook URL</mat-label>
              <input matInput formControlName="url" placeholder="https://example.com/hook" />
            </mat-form-field>
          </div>

          <div class="headers-section" formArrayName="headers">
            <div class="headers-title-row">
              <span class="headers-title">Headers</span>
              <button mat-stroked-button type="button" (click)="addHeaderRow()">
                <mat-icon>add</mat-icon>
                Add Header
              </button>
            </div>
            <div class="headers-hint">
              {{ task ? 'Optional. Add new headers only. Existing encrypted headers remain unchanged when left empty.' : 'Optional. Stored encrypted, never shown again.' }}
            </div>
            @for (row of headerRows.controls; track $index; let i = $index) {
              <div class="header-row" [formGroupName]="i">
                <mat-form-field appearance="outline" class="header-key-field">
                  <mat-label>Key</mat-label>
                  <input #headerKeyInput matInput formControlName="key" placeholder="Authorization" />
                </mat-form-field>
                <mat-form-field appearance="outline" class="header-value-field">
                  <mat-label>Value</mat-label>
                  <input matInput formControlName="value" placeholder="Bearer ..." />
                </mat-form-field>
                <button
                  mat-icon-button
                  type="button"
                  (click)="removeHeaderRow(i)"
                  [attr.aria-label]="'Remove header row ' + (i + 1)"
                >
                  <mat-icon>close</mat-icon>
                </button>
              </div>
            }
            @if (headerRows.errors?.['incomplete']) {
              <div class="headers-error">Fill both key and value for each header row.</div>
            }
            @if (headerRows.errors?.['duplicate']) {
              <div class="headers-error">Header keys must be unique.</div>
            }
            <mat-divider></mat-divider>
          </div>

          <mat-form-field appearance="outline" class="full-width">
            <mat-label>Body</mat-label>
            <textarea matInput formControlName="body" rows="3" placeholder="Optional request body"></textarea>
          </mat-form-field>

          <div class="toggle-row">
            <mat-slide-toggle formControlName="includeCertificate">
              Include certificate &amp; key in body
            </mat-slide-toggle>
          </div>
          @if (form.value.includeCertificate) {
            <mat-form-field appearance="outline" class="full-width">
              <mat-label>Certificate Format</mat-label>
              <mat-select formControlName="certificateFormat">
                @for (f of certificateFormats; track f) {
                  <mat-option [value]="f">{{ f.toUpperCase() }}</mat-option>
                }
              </mat-select>
              <mat-hint>PEM delivers individual files; PKCS#12 delivers a single encrypted archive.</mat-hint>
            </mat-form-field>
            <div class="cert-hint">
              On certificate renewal all four certificate files are sent to the webhook.
              Reference them in the body with
              <code>{{ certPlaceholder }}</code>,
              <code>{{ chainPlaceholder }}</code>,
              <code>{{ fullchainPlaceholder }}</code> and
              <code>{{ keyPlaceholder }}</code>
              . For PKCS#12 payloads use <code>{{ pkcs12Placeholder }}</code> or
              <code>{{ pkcs12Base64Placeholder }}</code>
              (append <code>_json</code> to any placeholder for safe JSON embedding).
              Leave the body empty to send the default JSON payload containing all
              selected format artifacts.
            </div>
          }

          <div class="toggle-row">
            <mat-slide-toggle formControlName="enabled">Enabled</mat-slide-toggle>
          </div>
        </form>
      </div>

      <div class="drawer-footer">
        <button mat-stroked-button type="button" (click)="drawer.close()">Cancel</button>
        <button mat-stroked-button type="button" [disabled]="form.invalid || testing" (click)="test()">
          {{ testing ? 'Testing…' : 'Test' }}
        </button>
        <button
          mat-flat-button
          color="primary"
          type="button"
          [disabled]="form.invalid || saving"
          (click)="save()"
        >
          {{ saving ? 'Saving…' : 'Save' }}
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
        height: 100%;
      }
      .drawer-header {
        flex-shrink: 0;
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
        padding: 16px 24px;
      }
      .drawer-footer {
        display: flex;
        justify-content: flex-end;
        gap: 8px;
        padding: 16px 24px 20px;
        border-top: 1px solid var(--launch-border-color);
        flex-shrink: 0;
      }
      .task-form {
        display: flex;
        flex-direction: column;
        gap: 4px;
      }
      .full-width {
        width: 100%;
      }
      .row {
        display: flex;
        gap: 8px;
      }
      .method-field {
        width: 120px;
        flex-shrink: 0;
      }
      .url-field {
        flex: 1;
      }
      .toggle-row {
        margin: 4px 0 8px;
      }
      .headers-section {
        display: flex;
        flex-direction: column;
        gap: 8px;
        margin-bottom: 10px;
      }
      .headers-title-row {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 8px;
      }
      .headers-title {
        font-size: 0.85rem;
        font-weight: 600;
        color: var(--launch-text-primary);
      }
      .headers-hint {
        font-size: 0.75rem;
        line-height: 1.4;
        color: var(--launch-text-muted);
      }
      .header-row {
        display: flex;
        align-items: flex-start;
        gap: 8px;
      }
      .header-key-field {
        flex: 0 0 34%;
      }
      .header-value-field {
        flex: 1;
      }
      .headers-error {
        font-size: 0.75rem;
        color: var(--mat-sys-error);
      }
      .cert-hint {
        font-size: 0.75rem;
        line-height: 1.5;
        color: var(--launch-text-muted);
        margin: 0 0 12px;
      }
      .cert-hint code {
        font-family: 'Roboto Mono', monospace;
        font-size: 0.7rem;
        background: rgba(255, 255, 255, 0.06);
        padding: 1px 4px;
        border-radius: 4px;
        color: var(--launch-text-secondary);
      }
      @media (max-width: 900px) {
        .header-row {
          flex-wrap: wrap;
        }
        .header-key-field,
        .header-value-field {
          flex: 1 1 100%;
        }
      }
    `,
  ],
})
export class TaskFormDrawerComponent {
  task: Task | null = null;
  recordId = 0;
  methods = HTTP_METHODS;
  certificateFormats = CERTIFICATE_FORMATS;
  saving = false;
  testing = false;
  readonly certPlaceholder = '{{cert}}';
  readonly chainPlaceholder = '{{chain}}';
  readonly fullchainPlaceholder = '{{fullchain}}';
  readonly keyPlaceholder = '{{private_key}}';
  readonly pkcs12Placeholder = '{{pkcs12}}';
  readonly pkcs12Base64Placeholder = '{{pkcs12_base64}}';
  readonly defaultPayloadExample = '{ "cert": …, "chain": …, "fullchain": …, "private_key": … }';

  @Output() onSave = new EventEmitter<void>();
  @ViewChild('drawer') drawer!: DrawerComponent;
  @ViewChildren('headerKeyInput') headerKeyInputs!: QueryList<ElementRef<HTMLInputElement>>;

  private readonly fb = inject(FormBuilder);
  private readonly api = inject(ApiService);
  private readonly notify = inject(NotifyService);

  form = this.fb.group({
    name: ['', Validators.required],
    triggerOn: [1, Validators.required],
    method: ['POST', Validators.required],
    url: ['', [Validators.required, Validators.pattern(/^https?:\/\/.+/i)]],
    headers: this.fb.array<FormGroup>([this.createHeaderRow()], { validators: [TaskFormDrawerComponent.validateHeaderRows] }),
    body: [''],
    includeCertificate: [false],
    certificateFormat: ['pem'],
    enabled: [true],
  });

  get headerRows(): FormArray<FormGroup> {
    return this.form.get('headers') as FormArray<FormGroup>;
  }

  addHeaderRow(key = '', value = ''): void {
    this.headerRows.push(this.createHeaderRow(key, value));
    setTimeout(() => {
      this.headerKeyInputs.last?.nativeElement.focus();
    });
  }

  removeHeaderRow(index: number): void {
    this.headerRows.removeAt(index);
    if (this.headerRows.length === 0) {
      this.addHeaderRow();
    }
  }

  open(recordId: number, task?: Task): void {
    this.recordId = recordId;
    this.task = task ?? null;
    if (task) {
      this.form.reset({
        name: task.name,
        triggerOn: task.triggerOn ?? 1,
        method: task.method || 'POST',
        url: task.url,
        body: task.body ?? '',
        includeCertificate: task.includeCertificate ?? false,
        certificateFormat: task.certificateFormat || 'pem',
        enabled: task.enabled,
      });
    } else {
      this.form.reset({
        name: '',
        triggerOn: 1,
        method: 'POST',
        url: '',
        body: '',
        includeCertificate: false,
        certificateFormat: 'pem',
        enabled: true,
      });
    }
    this.resetHeaderRows();
    this.drawer.open();
  }

  private buildPayload(): Task {
    const v = this.form.getRawValue();
    return {
      ...(this.task ?? {
        id: 0,
        createdAt: 0,
        updatedAt: 0,
        lastRun: 0,
        lastStatus: '',
        lastError: '',
      }),
      recordId: this.recordId,
      name: v.name!,
      triggerOn: v.triggerOn!,
      method: v.method!,
      url: v.url!,
      headers: this.serializeHeaders(),
      body: v.body ?? '',
      includeCertificate: v.includeCertificate ?? false,
      certificateFormat: (v.certificateFormat ?? 'pem') as string,
      enabled: v.enabled ?? true,
    };
  }

  test(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }
    this.testing = true;
    this.api.testTask(this.buildPayload()).subscribe({
      next: (res) => {
        this.testing = false;
        if (res.error) {
          this.notify.error(res.error, `Test failed (${res.status || 'no response'})`);
        } else {
          this.notify.message(`Test succeeded: ${res.status}`);
        }
      },
      error: (err) => {
        this.testing = false;
        this.notify.error(err?.error, 'Test failed');
      },
    });
  }

  save(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }
    this.saving = true;
    this.api.upsertTask(this.buildPayload()).subscribe({
      next: () => {
        this.saving = false;
        this.notify.message('Task saved.');
        this.onSave.emit();
        this.drawer.close();
      },
      error: (err) => {
        this.saving = false;
        this.notify.error(err?.error, 'Save failed');
      },
    });
  }

  private createHeaderRow(key = '', value = ''): FormGroup {
    return this.fb.group({
      key: [key],
      value: [value],
    });
  }

  private resetHeaderRows(): void {
    this.headerRows.clear();
    this.addHeaderRow();
    this.headerRows.markAsPristine();
  }

  private serializeHeaders(): string {
    const entries = this.headerRows.getRawValue() as HeaderRow[];
    const headers: Record<string, string> = {};

    for (const entry of entries) {
      const key = entry.key?.trim() ?? '';
      const value = entry.value?.trim() ?? '';
      if (!key && !value) {
        continue;
      }
      if (!key || !value) {
        continue;
      }
      headers[key] = value;
    }

    return Object.keys(headers).length ? JSON.stringify(headers) : '';
  }

  private static validateHeaderRows(control: AbstractControl): ValidationErrors | null {
    const rows = (control as FormArray<FormGroup>).getRawValue() as HeaderRow[];
    const usedKeys = new Set<string>();
    let hasIncomplete = false;
    let hasDuplicate = false;

    for (const row of rows) {
      const key = row.key?.trim() ?? '';
      const value = row.value?.trim() ?? '';

      if (!key && !value) {
        continue;
      }

      if (!key || !value) {
        hasIncomplete = true;
        continue;
      }

      const normalized = key.toLowerCase();
      if (usedKeys.has(normalized)) {
        hasDuplicate = true;
      }
      usedKeys.add(normalized);
    }

    if (!hasIncomplete && !hasDuplicate) {
      return null;
    }

    return {
      ...(hasIncomplete ? { incomplete: true } : {}),
      ...(hasDuplicate ? { duplicate: true } : {}),
    };
  }
}
