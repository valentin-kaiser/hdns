import { CommonModule } from '@angular/common';
import { Component, OnDestroy, OnInit, ViewChild, inject, signal } from '@angular/core';
import { ReactiveFormsModule } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTooltipModule } from '@angular/material/tooltip';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { Subscription } from 'rxjs';
import { CertificateDetails, Record as DnsRecord, RecordPurpose, Resolution, Task, TaskTrigger } from '../../global/model/api';
import { ApiService, Stream } from '../../global/services/api/api.service';
import { NotifyService } from '../../global/services/notify/notify.service';
import { RecordFormDrawerComponent } from '../home/drawers/record-form/record-form-drawer.component';
import { TaskFormDrawerComponent } from '../home/drawers/task-form/task-form-drawer.component';

@Component({
  selector: 'app-record-overview',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    RouterLink,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatTooltipModule,
    RecordFormDrawerComponent,
    TaskFormDrawerComponent,
  ],
  template: `
    <div class="overview-page">
      <div class="hdns-card overview-card">
        <div class="header-row">
          <div>
            @if (record()) {
              <h2>{{ record()!.name }}.{{ record()!.domain }}</h2>
            } @else {
              <h2>Record Overview</h2>
            }
            <p class="subtitle mono">Current IP: {{ currentIp() || '—' }}</p>
          </div>
          <div class="header-actions">
            <button mat-stroked-button routerLink="/">
              <mat-icon>arrow_back</mat-icon>
              Back to records
            </button>
            @if (record()) {
              <button mat-flat-button color="primary" type="button" (click)="openEditDrawer()">
                <mat-icon>edit</mat-icon>
                Edit
              </button>
            }
          </div>
        </div>

        @if (loadingRecord()) {
          <div class="center"><mat-spinner diameter="24"></mat-spinner> Loading record…</div>
        } @else if (!record()) {
          <div class="center muted">Record not found.</div>
        } @else {
          <div class="layout-grid">
            @if (recordDoesDdns(record()!)) {
              <section class="section card resolve-card resolve-full">
                <div class="section-title-row">
                  <h3>Resolve</h3>
                </div>
                @if (resolveResults().length === 0) {
                  <div class="resolve-empty-state">
                    <mat-icon class="resolve-empty-icon">travel_explore</mat-icon>
                    <div class="resolve-empty-title">Waiting for results</div>
                    <div class="resolve-empty-sub">
                      DNS servers will appear here as they respond.
                    </div>
                  </div>
                } @else {
                  <div class="resolve-list">
                    @for (r of resolveResults(); track r.server) {
                      <div
                        class="resolve-row"
                        [class.has-error]="!!r.error"
                        [class.has-success]="!r.error && !!r.addresses?.length"
                      >
                        <div class="resolve-top-row">
                          <div class="resolve-server-col">
                            <span
                              class="resolve-status-dot"
                              [class.dot-ok]="!r.error && !!r.addresses?.length"
                              [class.dot-err]="!!r.error"
                            ></span>
                            <div class="resolve-server-meta">
                              <div class="resolve-server-label">DNS Server</div>
                              <div class="resolve-server-name">{{ r.server }}</div>
                            </div>
                          </div>
                          @if (r.responseTime) {
                            <div class="resolve-response-pill" [class.pill-err]="!!r.error">
                              <mat-icon class="resolve-pill-icon">schedule</mat-icon>
                              <span>{{ r.responseTime }}ms</span>
                            </div>
                          }
                        </div>

                        <div class="resolve-content-col">
                          @if (r.addresses?.length && !r.error) {
                            <div class="resolve-address-list">
                              @for (a of r.addresses; track a) {
                                <div class="resolve-address-row">{{ a }}</div>
                              }
                            </div>
                          }

                          @if (r.error) {
                            <div class="resolve-error-row">
                              <mat-icon class="resolve-err-icon">error_outline</mat-icon>
                              <span>{{ r.error }}</span>
                            </div>
                          }

                          @if (!r.error && !r.addresses?.length) {
                            <div class="resolve-pending-msg">No addresses returned.</div>
                          }
                        </div>
                      </div>
                    }
                  </div>
                }
              </section>
            }

            @if (recordHasCert(record()!)) {
              <section class="section card">
                <div class="section-title-row">
                  <h3>Certificate</h3>
                  <div class="row-actions">
                    @if (certificateDetails()?.certificate) {
                      <span
                        class="status"
                        [class]="statusClass(certificateDetails()!.certificate!.status)"
                      >
                        <mat-icon>{{
                          statusIcon(certificateDetails()!.certificate!.status)
                        }}</mat-icon>
                        {{ certificateDetails()!.certificate!.status }}
                      </span>
                    }
                  </div>
                </div>

                @if (certificateDetails()?.certificate) {
                  <div class="meta-grid">
                    <div class="meta-item">
                      <span class="label">Domains</span>
                      <span class="value mono">{{
                        certificateDetails()!.certificate!.domains || '—'
                      }}</span>
                    </div>
                    <div class="meta-item">
                      <span class="label">Serial</span>
                      <span class="value mono">{{
                        certificateDetails()!.certificate!.serial || '—'
                      }}</span>
                    </div>
                    <div class="meta-item">
                      <span class="label">Valid Until</span>
                      <span class="value">{{
                        certificateDetails()!.certificate!.notAfter
                          ? (certificateDetails()!.certificate!.notAfter
                            | date: 'dd.MM.yyyy HH:mm' : 'UTC')
                          : '—'
                      }}</span>
                    </div>
                    @if (certificateDetails()!.certificate!.lastError) {
                      <div class="meta-item full">
                        <span class="label">Last Error</span>
                        <span class="value err">{{
                          certificateDetails()!.certificate!.lastError
                        }}</span>
                      </div>
                    }
                  </div>
                } @else {
                  <p class="muted">No certificate has been issued yet.</p>
                }

                @if (certificateDetails()) {
                  <div class="download-grid">
                    @for (artifact of certificateDetails()!.artifacts; track artifact.key) {
                      <button
                        mat-stroked-button
                        type="button"
                        class="download-btn"
                        [disabled]="!artifact.available"
                        [matTooltip]="artifact.available ? '' : 'Artifact is not available yet'"
                        (click)="download(artifact.key)"
                      >
                        <mat-icon>download</mat-icon>
                        {{ artifact.label }}
                      </button>
                    }
                  </div>
                }
              </section>
            }

            <section class="section card">
              <div class="section-title-row">
                <h3>Tasks</h3>
                <button mat-flat-button color="primary" (click)="addTask()">
                  <mat-icon>add</mat-icon>
                  Add task
                </button>
              </div>
              @if (tasks().length === 0) {
                <p class="muted">No tasks yet for this record.</p>
              } @else {
                <div class="task-list">
                  @for (t of tasks(); track t.id) {
                    <div class="task-item">
                      <div class="task-main">
                        <div class="task-title">
                          <span class="task-name">{{ t.name }}</span>
                          <span class="chip">{{ triggerLabel(t.triggerOn) }}</span>
                        </div>
                        <div class="mono">{{ t.method }} {{ t.url }}</div>
                        @if (+t.lastRun) {
                          <div class="history-meta">
                            Last run {{ t.lastRun | date: 'dd.MM.yyyy HH:mm' : 'UTC' }}
                            @if (t.lastStatus) {
                              · {{ t.lastStatus }}
                            }
                            @if (t.lastError) {
                              · <span class="err">{{ t.lastError }}</span>
                            }
                          </div>
                        }
                      </div>
                      <div class="task-actions">
                        <button mat-icon-button matTooltip="Run" (click)="runTask(t)">
                          <mat-icon>play_arrow</mat-icon>
                        </button>
                        <button mat-icon-button matTooltip="Edit" (click)="editTask(t)">
                          <mat-icon>edit</mat-icon>
                        </button>
                        <button mat-icon-button matTooltip="Delete" (click)="removeTask(t)">
                          <mat-icon>delete</mat-icon>
                        </button>
                      </div>
                    </div>
                  }
                </div>
              }
            </section>

            <section class="section card">
              <h3>Task History</h3>
              @if (!certificateDetails() || certificateDetails()!.taskRuns.length === 0) {
                <p class="muted">No certificate-triggered task runs yet.</p>
              } @else {
                <div class="history-list">
                  @for (run of certificateDetails()!.taskRuns; track run.id) {
                    <div class="history-row">
                      <div class="history-head">
                        <span class="chip">{{ run.taskName || 'Task #' + run.taskId }}</span>
                        <span
                          class="chip"
                          [class.chip-ok]="run.status === 'success'"
                          [class.chip-err]="run.status === 'failed'"
                          >{{ run.status }}</span
                        >
                      </div>
                      <div class="history-meta">
                        {{ run.startedAt | date: 'dd.MM.yyyy HH:mm:ss' : 'UTC' }}
                        @if (run.durationMs) {
                          · {{ formatDuration(run.durationMs) }}
                        }
                      </div>
                      @if (run.error) {
                        <div class="err">{{ run.error }}</div>
                      }
                    </div>
                  }
                </div>
              }
            </section>

            @if (recordHasCert(record()!)) {
              <section class="section card">
                <h3>Issuance History</h3>
                @if (!certificateDetails() || certificateDetails()!.issuanceJobs.length === 0) {
                  <p class="muted">No issuance jobs yet.</p>
                } @else {
                  <div class="history-list">
                    @for (job of certificateDetails()!.issuanceJobs; track job.id) {
                      <div class="history-row">
                        <div class="history-head">
                          <span class="chip">{{ job.source }}</span>
                          <span
                            class="chip"
                            [class.chip-ok]="job.status === 'success'"
                            [class.chip-err]="job.status === 'failed'"
                            >{{ job.status }}</span
                          >
                        </div>
                        <div class="history-meta">
                          {{ job.startedAt | date: 'dd.MM.yyyy HH:mm:ss' : 'UTC' }}
                          @if (job.durationMs) {
                            · {{ formatDuration(job.durationMs) }}
                          }
                        </div>
                        @if (job.error) {
                          <div class="err">{{ job.error }}</div>
                        }
                      </div>
                    }
                  </div>
                }
              </section>
            }
          </div>
        }
      </div>
    </div>

    <app-record-form-drawer #recordDrawer (onSave)="reloadRecord()" />
    <app-task-form-drawer #taskForm (onSave)="loadTasksAndCertificate()" />
  `,
  styles: [
    `
      .overview-page {
        width: 100%;
        max-width: 1480px;
        margin: 0 auto;
        padding: 16px 20px;
        box-sizing: border-box;
      }
      .overview-card {
        padding: 20px 24px;
      }
      .header-row {
        display: flex;
        align-items: flex-start;
        justify-content: space-between;
        gap: 12px;
        flex-wrap: wrap;
        margin-bottom: 14px;
      }
      .header-actions {
        display: flex;
        align-items: center;
        gap: 8px;
        flex-wrap: wrap;
      }
      .subtitle {
        margin: 4px 0 0;
        color: var(--launch-text-muted);
      }
      .layout-grid {
        display: grid;
        grid-template-columns: 1fr;
        gap: 12px;
      }
      .left-stack {
        display: flex;
        flex-direction: column;
        gap: 12px;
      }
      .resolve-full {
        width: 100%;
      }
      .section.card {
        border: 1px solid var(--launch-border-color);
        border-radius: 10px;
        background: rgba(255, 255, 255, 0.02);
        padding: 12px;
      }
      .section h3 {
        margin: 0 0 8px;
        font-size: 0.95rem;
        color: var(--launch-text-primary);
      }
      .section-title-row {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 8px;
        margin-bottom: 8px;
      }
      .meta-grid {
        display: grid;
        grid-template-columns: repeat(2, minmax(0, 1fr));
        gap: 10px;
      }
      .meta-item {
        display: flex;
        flex-direction: column;
        gap: 2px;
      }
      .meta-item.full {
        grid-column: 1 / -1;
      }
      .label {
        font-size: 0.6875rem;
        color: var(--launch-text-muted);
        text-transform: uppercase;
        letter-spacing: 0.05em;
      }
      .value {
        font-size: 0.8125rem;
        color: var(--launch-text-secondary);
      }
      .download-grid {
        margin-top: 10px;
        display: grid;
        grid-template-columns: repeat(2, minmax(0, 1fr));
        gap: 8px;
      }
      .download-btn {
        justify-content: flex-start;
      }
      .history-list,
      .task-list {
        display: flex;
        flex-direction: column;
        gap: 8px;
      }
      .history-row,
      .task-item {
        border: 1px solid var(--launch-border-color);
        border-radius: 8px;
        padding: 10px;
        display: flex;
        flex-direction: column;
        gap: 6px;
      }
      .task-item {
        flex-direction: row;
        justify-content: space-between;
        align-items: flex-start;
      }
      .task-main {
        min-width: 0;
        flex: 1;
      }
      .task-title,
      .history-head {
        display: flex;
        align-items: center;
        gap: 6px;
        flex-wrap: wrap;
      }
      .task-name {
        font-weight: 600;
        color: var(--launch-text-primary);
      }
      .task-actions {
        display: flex;
      }
      .chip {
        font-size: 0.6875rem;
        border: 1px solid var(--launch-border-color);
        border-radius: 999px;
        padding: 1px 8px;
        color: var(--launch-text-secondary);
      }
      .chip-ok,
      .status-ok {
        color: var(--hdns-success, #2e7d32);
      }
      .chip-err,
      .status-err,
      .err {
        color: var(--hdns-danger, #d32f2f);
      }
      .status {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        font-size: 0.75rem;
        text-transform: capitalize;
      }
      .status mat-icon {
        font-size: 16px;
        width: 16px;
        height: 16px;
      }
      .status-warn {
        color: var(--hdns-warning, #ed6c02);
      }
      .history-meta,
      .muted {
        font-size: 0.75rem;
        color: var(--launch-text-muted);
      }
      .row-actions {
        display: flex;
        gap: 8px;
        align-items: center;
        flex-wrap: wrap;
      }
      .mono {
        font-family: 'Roboto Mono', monospace;
        font-size: 0.8125rem;
      }
      .center {
        display: flex;
        align-items: center;
        gap: 10px;
        color: var(--launch-text-muted);
      }

      .resolve-card {
        padding: 0;
        overflow: hidden;
      }
      .resolve-card .section-title-row {
        padding: 12px;
        border-bottom: 1px solid var(--launch-border-color);
        margin-bottom: 0;
      }
      .resolve-empty-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 6px;
        padding: 32px 16px;
        color: var(--launch-text-muted);
        text-align: center;
      }
      .resolve-empty-icon {
        font-size: 36px;
        width: 36px;
        height: 36px;
        opacity: 0.6;
      }
      .resolve-empty-title {
        font-size: 0.95rem;
        font-weight: 600;
        color: var(--launch-text-secondary);
      }
      .resolve-empty-sub {
        font-size: 0.8125rem;
      }
      .resolve-list {
        display: grid;
        grid-template-columns: repeat(4, minmax(0, 1fr));
        gap: 10px;
        padding: 12px;
      }
      .resolve-row {
        display: flex;
        flex-direction: column;
        background: #16161e;
        border: 1px solid var(--launch-border-color);
        border-radius: 8px;
        overflow: hidden;
        min-width: 0;
      }
      .resolve-row.has-success {
        border-color: color-mix(in srgb, var(--hdns-success) 40%, var(--launch-border-color));
      }
      .resolve-row.has-error {
        border-color: color-mix(in srgb, var(--hdns-danger) 50%, var(--launch-border-color));
      }
      .resolve-top-row {
        display: flex;
        align-items: flex-start;
        justify-content: space-between;
        gap: 8px;
        padding: 10px 12px;
        border-bottom: 1px solid var(--launch-border-color);
        background: rgba(255, 255, 255, 0.02);
      }
      .resolve-server-col {
        display: flex;
        align-items: center;
        gap: 10px;
        min-width: 0;
      }
      .resolve-status-dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        background: var(--launch-text-muted);
        box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.04);
        flex-shrink: 0;
      }
      .resolve-status-dot.dot-ok {
        background: var(--hdns-success);
        box-shadow: 0 0 0 3px var(--hdns-primary-tint);
      }
      .resolve-status-dot.dot-err {
        background: var(--hdns-danger);
        box-shadow: 0 0 0 3px color-mix(in srgb, var(--hdns-danger) 20%, transparent);
      }
      .resolve-server-meta {
        min-width: 0;
        display: flex;
        flex-direction: column;
        gap: 2px;
      }
      .resolve-server-label {
        font-size: 0.6875rem;
        text-transform: uppercase;
        letter-spacing: 0.06em;
        color: var(--launch-text-muted);
        font-weight: 500;
      }
      .resolve-server-name {
        font-family: 'Roboto Mono', monospace;
        font-size: 0.8125rem;
        font-weight: 600;
        color: var(--launch-text-primary);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
      }
      .resolve-content-col {
        padding: 10px 12px;
        display: flex;
        flex-direction: column;
        gap: 8px;
        min-width: 0;
      }
      .resolve-response-pill {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        padding: 2px 8px;
        border-radius: 999px;
        background: rgba(255, 255, 255, 0.04);
        border: 1px solid var(--launch-border-color);
        font-size: 0.6875rem;
        color: var(--launch-text-muted);
        font-variant-numeric: tabular-nums;
      }
      .resolve-pill-icon {
        font-size: 12px;
        width: 12px;
        height: 12px;
      }
      .resolve-address-list {
        display: flex;
        flex-direction: column;
        gap: 4px;
      }
      .resolve-address-row {
        font-family: 'Roboto Mono', monospace;
        font-size: 0.8125rem;
        color: var(--hdns-success);
        padding: 4px 8px;
        border-radius: 4px;
        word-break: break-all;
      }
      .resolve-error-row {
        display: flex;
        align-items: flex-start;
        gap: 6px;
        font-size: 0.8125rem;
        color: var(--hdns-danger);
      }
      .resolve-err-icon {
        font-size: 16px;
        width: 16px;
        height: 16px;
        margin-top: 1px;
        flex-shrink: 0;
      }
      .resolve-pending-msg {
        font-size: 0.8125rem;
        color: var(--launch-text-muted);
        font-style: italic;
      }
      @media (max-width: 1600px) {
        .resolve-list {
          grid-template-columns: repeat(3, minmax(0, 1fr));
        }
      }
      @media (max-width: 1250px) {
        .resolve-list {
          grid-template-columns: repeat(2, minmax(0, 1fr));
        }
      }
      @media (max-width: 1000px) {
        .resolve-list {
          grid-template-columns: 1fr;
        }
        .download-grid,
        .meta-grid {
          grid-template-columns: 1fr;
        }
      }
    `,
  ],
})
export class RecordOverviewComponent implements OnInit, OnDestroy {
  readonly record = signal<DnsRecord | null>(null);
  readonly loadingRecord = signal(true);
  readonly currentIp = signal('');
  readonly tasks = signal<Task[]>([]);
  readonly certificateDetails = signal<CertificateDetails | null>(null);
  readonly resolveResults = signal<Resolution[]>([]);
  readonly resolving = signal(false);

  @ViewChild('recordDrawer') recordDrawer!: RecordFormDrawerComponent;
  @ViewChild('taskForm') taskForm!: TaskFormDrawerComponent;

  private readonly api = inject(ApiService);
  private readonly notify = inject(NotifyService);
  private readonly route = inject(ActivatedRoute);

  private stream: Stream<Resolution, DnsRecord> | null = null;
  private messagesSub?: Subscription;
  private connectedSub?: Subscription;

  ngOnInit(): void {
    this.loadCurrentIp();

    const recordId = Number(this.route.snapshot.paramMap.get('recordId'));
    if (!Number.isFinite(recordId) || recordId <= 0) {
      this.loadingRecord.set(false);
      this.notify.error('Invalid record id', 'Overview');
      return;
    }

    this.loadRecord(recordId);
  }

  ngOnDestroy(): void {
    this.stopResolve();
  }

  openEditDrawer(): void {
    const record = this.record();
    if (!record) return;
    this.recordDrawer.open(record);
  }

  reloadRecord(): void {
    const record = this.record();
    if (record) {
      this.loadRecord(record.id);
    }
  }

  private loadCurrentIp(): void {
    this.api.getAddress().subscribe({
      next: (addr) => this.currentIp.set(addr?.ipv4 || addr?.ipv6 || ''),
      error: () => this.currentIp.set(''),
    });
  }

  private loadRecord(recordId: number): void {
    this.loadingRecord.set(true);
    this.api.getRecords({ search: '' }).subscribe({
      next: (res) => {
        const found = (res.records ?? []).find((r) => Number(r.id) === recordId) ?? null;
        this.record.set(found);
        this.loadingRecord.set(false);
        if (!found) {
          return;
        }
        this.loadTasksAndCertificate();
        if (this.recordDoesDdns(found)) {
          this.startResolve();
        }
      },
      error: (err) => {
        this.loadingRecord.set(false);
        this.notify.error(err?.error, 'Failed to load record');
      },
    });
  }

  loadTasksAndCertificate(): void {
    const record = this.record();
    if (!record) return;

    this.api.getTasks().subscribe({
      next: (res) => {
        this.tasks.set(
          (res.tasks ?? []).filter(
            (t) => t.recordId === record.id && this.taskMatchesRecordPurpose(t, record)
          )
        );
      },
      error: (err) => this.notify.error(err?.error, 'Failed to load tasks'),
    });

    this.api.getCertificateDetails(record).subscribe({
      next: (details) => this.certificateDetails.set(details),
      error: (err) => this.notify.error(err?.error, 'Failed to load certificate details'),
    });
  }

  addTask(): void {
    const record = this.record();
    if (!record) return;
    this.taskForm.openForRecord(record.id, record.purpose ?? RecordPurpose.RECORD_PURPOSE_UNSPECIFIED);
  }

  editTask(task: Task): void {
    const record = this.record();
    if (!record) return;
    this.taskForm.openForRecord(record.id, record.purpose ?? RecordPurpose.RECORD_PURPOSE_UNSPECIFIED, task);
  }

  runTask(task: Task): void {
    this.api.runTask(task).subscribe({
      next: (res) => {
        if (res.error) {
          this.notify.error(res.error, `Run failed (${res.status || 'no response'})`);
        } else {
          this.notify.message(`Run succeeded: ${res.status}`);
        }
        this.loadTasksAndCertificate();
      },
      error: (err) => this.notify.error(err?.error, 'Run failed'),
    });
  }

  removeTask(task: Task): void {
    this.notify.warning({
      title: 'Delete task',
      message: `Delete task "${task.name}"? This cannot be undone.`,
      buttons: [
        { text: 'Cancel', color: 'accent' },
        { text: 'Delete', color: 'warn', handler: () => this.deleteTask(task.id) },
      ],
    });
  }

  private deleteTask(taskId: number): void {
    this.api.deleteTask(taskId).subscribe({
      next: () => {
        this.notify.message('Task deleted.');
        this.loadTasksAndCertificate();
      },
      error: (err) => this.notify.error(err?.error, 'Delete failed'),
    });
  }

  startResolve(): void {
    const record = this.record();
    if (!record || !this.recordDoesDdns(record)) {
      return;
    }

    this.stopResolve();
    this.resolveResults.set([]);
    this.resolving.set(true);

    this.stream = this.api.streamResolveRecord();
    this.connectedSub = this.stream.connect$.subscribe(() => {
      this.stream?.send(record);
    });

    this.messagesSub = this.stream.messages$.subscribe({
      next: (msg) => {
        const next = [...this.resolveResults().filter((r) => r.server !== msg.server), msg];
        this.resolveResults.set(next);
      },
      error: (err) => {
        this.notify.error(err?.message ?? String(err), 'Resolve stream error');
        this.stopResolve();
      },
    });
  }

  stopResolve(): void {
    this.messagesSub?.unsubscribe();
    this.messagesSub = undefined;
    this.connectedSub?.unsubscribe();
    this.connectedSub = undefined;
    this.stream?.close();
    this.stream = null;
    this.resolving.set(false);
  }

  download(artifact: string): void {
    const record = this.record();
    if (!record) return;
    const url = this.api.getCertificateDownloadUrl(record.id, artifact);
    window.open(url, '_blank', 'noopener');
  }

  triggerLabel(trigger: number): string {
    switch (trigger) {
      case 1:
        return 'On IP update';
      case 2:
        return 'On cert renewal';
      case 3:
        return 'On IP & cert';
      default:
        return 'Unknown';
    }
  }

  formatDuration(durationMs: number): string {
    if (!durationMs || durationMs <= 0) return '0s';
    if (durationMs < 1000) return `${durationMs}ms`;
    return `${(durationMs / 1000).toFixed(1)}s`;
  }

  statusClass(status: string): string {
    switch (status) {
      case 'valid':
      case 'success':
        return 'status-ok';
      case 'failed':
        return 'status-err';
      default:
        return 'status-warn';
    }
  }

  statusIcon(status: string): string {
    switch (status) {
      case 'valid':
      case 'success':
        return 'verified';
      case 'failed':
        return 'error';
      default:
        return 'hourglass_empty';
    }
  }

  recordHasCert(record: DnsRecord): boolean {
    const purpose = record.purpose ?? RecordPurpose.RECORD_PURPOSE_UNSPECIFIED;
    return (
      purpose === RecordPurpose.RECORD_PURPOSE_CERT || purpose === RecordPurpose.RECORD_PURPOSE_BOTH
    );
  }

  recordDoesDdns(record: DnsRecord): boolean {
    const purpose = record.purpose ?? RecordPurpose.RECORD_PURPOSE_UNSPECIFIED;
    return (
      purpose === RecordPurpose.RECORD_PURPOSE_DDNS || purpose === RecordPurpose.RECORD_PURPOSE_BOTH
    );
  }

  private taskMatchesRecordPurpose(task: Task, record: DnsRecord): boolean {
    const purpose = record.purpose ?? RecordPurpose.RECORD_PURPOSE_UNSPECIFIED;
    switch (purpose) {
      case RecordPurpose.RECORD_PURPOSE_DDNS:
        return task.triggerOn === TaskTrigger.TASK_TRIGGER_IP;
      case RecordPurpose.RECORD_PURPOSE_CERT:
        return task.triggerOn === TaskTrigger.TASK_TRIGGER_CERT;
      default:
        return true;
    }
  }
}
