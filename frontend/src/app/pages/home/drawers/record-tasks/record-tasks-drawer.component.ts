import { CommonModule } from '@angular/common';
import { Component, inject, signal, ViewChild } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatTooltipModule } from '@angular/material/tooltip';
import { DrawerComponent } from '../../../../components/drawer/drawer.component';
import { Record as DnsRecord, Task } from '../../../../global/model/api';
import { ApiService } from '../../../../global/services/api/api.service';
import { NotifyService } from '../../../../global/services/notify/notify.service';
import { TaskFormDrawerComponent } from '../task-form/task-form-drawer.component';

@Component({
  selector: 'app-record-tasks-drawer',
  standalone: true,
  imports: [
    CommonModule,
    MatButtonModule,
    MatIconModule,
    MatTooltipModule,
    DrawerComponent,
    TaskFormDrawerComponent,
  ],
  template: `
    <app-drawer #drawer (closed)="drawer.close()" [width]="45" [breakpoints]="[{ maxWidth: 768, width: 100 }, { maxWidth: 1200, width: 80 }, { maxWidth: 1600, width: 60 }]">
      <div class="drawer-header">
        <h3 class="drawer-title">Tasks</h3>
        @if (record) {
          <span class="drawer-subtitle">{{ record.name }}.{{ record.domain }}</span>
        }
      </div>

      <div class="drawer-body">
        <div class="toolbar">
          <span class="hint">Webhooks executed on IP update or certificate renewal.</span>
          <button mat-flat-button color="primary" (click)="add()">
            <mat-icon>add</mat-icon> Add task
          </button>
        </div>

        @if (loading()) {
          <div class="empty">Loading…</div>
        } @else if (tasks().length === 0) {
          <div class="empty">
            <mat-icon>webhook</mat-icon>
            <p>No tasks yet for this record.</p>
          </div>
        } @else {
          <div class="task-list">
            @for (t of tasks(); track t.id) {
              <div class="task-item">
                <div class="task-main">
                  <div class="task-title">
                    <span class="task-name">{{ t.name }}</span>
                    <span class="badge" [class.badge-on]="t.enabled" [class.badge-off]="!t.enabled">
                      {{ t.enabled ? 'Enabled' : 'Disabled' }}
                    </span>
                    <span class="badge badge-trigger">{{ triggerLabel(t.triggerOn) }}</span>
                  </div>
                  <div class="task-url mono">{{ t.method }} {{ t.url }}</div>
                  @if (t.lastRun) {
                    <div class="task-meta">
                      Last run {{ t.lastRun | date: 'dd.MM.yyyy HH:mm' : 'UTC' }}
                      @if (t.lastStatus) { · {{ t.lastStatus }} }
                      @if (t.lastError) { · <span class="err">{{ t.lastError }}</span> }
                    </div>
                  }
                </div>
                <div class="task-actions">
                  <button mat-icon-button matTooltip="Test" (click)="testTask(t)">
                    <mat-icon>play_arrow</mat-icon>
                  </button>
                  <button mat-icon-button matTooltip="Edit" (click)="edit(t)">
                    <mat-icon>edit</mat-icon>
                  </button>
                  <button mat-icon-button matTooltip="Delete" class="danger" (click)="remove(t)">
                    <mat-icon>delete</mat-icon>
                  </button>
                </div>
              </div>
            }
          </div>
        }
      </div>

      <div class="drawer-footer">
        <button mat-stroked-button (click)="drawer.close()">Close</button>
      </div>
    </app-drawer>

    <app-task-form-drawer #taskForm (onSave)="load()" />
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
        display: flex;
        align-items: baseline;
        gap: 10px;
      }
      .drawer-title {
        margin: 0;
        font-size: 1rem;
        font-weight: 600;
        color: var(--launch-text-primary);
      }
      .drawer-subtitle {
        font-family: 'Roboto Mono', monospace;
        font-size: 0.8125rem;
        color: var(--launch-text-muted);
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
      .toolbar {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 12px;
        margin-bottom: 16px;
      }
      .hint {
        font-size: 0.8125rem;
        color: var(--launch-text-muted);
      }
      .empty {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 8px;
        padding: 40px 16px;
        color: var(--launch-text-muted);
        font-size: 0.875rem;
      }
      .empty mat-icon {
        font-size: 36px;
        width: 36px;
        height: 36px;
      }
      .task-list {
        display: flex;
        flex-direction: column;
        gap: 8px;
      }
      .task-item {
        display: flex;
        align-items: flex-start;
        justify-content: space-between;
        gap: 8px;
        padding: 12px;
        border: 1px solid var(--launch-border-color);
        border-radius: 8px;
        background: rgba(255, 255, 255, 0.02);
      }
      .task-main {
        min-width: 0;
        flex: 1;
      }
      .task-title {
        display: flex;
        align-items: center;
        gap: 8px;
        flex-wrap: wrap;
      }
      .task-name {
        font-weight: 600;
        color: var(--launch-text-primary);
        font-size: 0.9rem;
      }
      .badge {
        font-size: 0.6875rem;
        font-weight: 600;
        padding: 1px 8px;
        border-radius: 999px;
      }
      .badge-on {
        color: var(--hdns-primary);
        background: var(--hdns-primary-tint);
      }
      .badge-off {
        color: var(--launch-text-muted);
        background: rgba(255, 255, 255, 0.06);
      }
      .badge-trigger {
        color: var(--hdns-warning, #ed6c02);
        background: rgba(237, 108, 2, 0.12);
      }
      .task-url {
        margin-top: 4px;
        color: var(--launch-text-secondary);
        word-break: break-all;
      }
      .mono {
        font-family: 'Roboto Mono', monospace;
        font-size: 0.8125rem;
      }
      .task-meta {
        margin-top: 4px;
        font-size: 0.75rem;
        color: var(--launch-text-muted);
      }
      .err {
        color: var(--hdns-danger, #d32f2f);
      }
      .task-actions {
        display: flex;
        flex-shrink: 0;
      }
      .danger mat-icon {
        color: var(--hdns-danger, #d32f2f);
      }
    `,
  ],
})
export class RecordTasksDrawerComponent {
  record: DnsRecord | null = null;
  readonly tasks = signal<Task[]>([]);
  readonly loading = signal(false);

  @ViewChild('drawer') drawer!: DrawerComponent;
  @ViewChild('taskForm') taskForm!: TaskFormDrawerComponent;

  private readonly api = inject(ApiService);
  private readonly notify = inject(NotifyService);

  open(record: DnsRecord): void {
    this.record = record;
    this.drawer.open();
    this.load();
  }

  load(): void {
    if (!this.record) return;
    const recordId = this.record.id;
    this.loading.set(true);
    this.api.getTasks().subscribe({
      next: (res) => {
        this.tasks.set((res.tasks ?? []).filter((t) => t.recordId === recordId));
        this.loading.set(false);
      },
      error: (err) => {
        this.loading.set(false);
        this.notify.error(err?.error, 'Failed to load tasks');
      },
    });
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

  add(): void {
    if (!this.record) return;
    this.taskForm.open(this.record.id);
  }

  edit(task: Task): void {
    if (!this.record) return;
    this.taskForm.open(this.record.id, task);
  }

  testTask(task: Task): void {
    this.api.testTask(task).subscribe({
      next: (res) => {
        if (res.error) {
          this.notify.error(res.error, `Test failed (${res.status || 'no response'})`);
        } else {
          this.notify.message(`Test succeeded: ${res.status}`);
        }
        this.load();
      },
      error: (err) => this.notify.error(err?.error, 'Test failed'),
    });
  }

  remove(task: Task): void {
    this.notify.warning({
      title: 'Delete task',
      message: `Delete task "${task.name}"? This cannot be undone.`,
      buttons: [
        { text: 'Cancel', color: 'accent' },
        {
          text: 'Delete',
          color: 'warn',
          handler: () => this.deleteTask(task),
        },
      ],
    });
  }

  private deleteTask(task: Task): void {
    this.api.deleteTask(task.id).subscribe({
      next: () => {
        this.notify.message('Task deleted.');
        this.load();
      },
      error: (err) => this.notify.error(err?.error, 'Delete failed'),
    });
  }
}
