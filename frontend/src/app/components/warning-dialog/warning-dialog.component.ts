import { Component, Inject } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material/dialog';
import { Warning } from '../../global/services/notify/notify.model';

@Component({
  selector: 'app-warning-dialog',
  standalone: true,
  imports: [MatDialogModule, MatButtonModule],
  template: `
    <div class="dialog-header">
      <span class="material-symbols-outlined header-icon">warning</span>
      <span class="header-title">{{ data.title }}</span>
    </div>

    <mat-dialog-content class="dialog-body">
      <p>{{ data.message }}</p>
    </mat-dialog-content>

    <mat-dialog-actions align="end" class="dialog-footer">
      @for (btn of data.buttons; track btn.text) {
        <button mat-stroked-button
                [class.is-destructive]="btn.color === 'warn'"
                [class.is-primary]="btn.color === 'primary'"
                (click)="handleButton(btn.handler)">
          {{ btn.text }}
        </button>
      }
    </mat-dialog-actions>
  `,
  styles: [`
    .dialog-header {
      display: flex;
      align-items: center;
      gap: 10px;
      padding: 20px 24px 14px;
      border-bottom: 1px solid var(--launch-border-color);
    }

    .header-icon {
      font-size: 20px;
      line-height: 1;
      color: var(--launch-warning-icon);
      flex-shrink: 0;
    }

    .header-title {
      font-size: 0.9375rem;
      font-weight: 600;
      color: var(--launch-text-primary);
      line-height: 1.4;
    }

    .dialog-body {
      padding: 16px 24px !important;

      p {
        margin: 0;
        font-size: 0.875rem;
        color: var(--launch-text-secondary);
        line-height: 1.65;
      }
    }

    .dialog-footer {
      padding: 8px 16px 12px !important;
      border-top: 1px solid var(--launch-border-color);
      gap: 8px;
      margin: 0 !important;
    }

    .is-destructive {
      color: var(--launch-error) !important;
      border-color: var(--launch-error) !important;
    }

    .is-primary {
      background-color: var(--launch-accent) !important;
      border-color: var(--launch-accent) !important;
      color: #0f172a !important;
    }
  `],
})
export class WarningDialogComponent {
  constructor(
    public dialogRef: MatDialogRef<WarningDialogComponent>,
    @Inject(MAT_DIALOG_DATA) public data: Warning,
  ) {}

  handleButton(handler?: () => void): void {
    handler?.();
    this.dialogRef.close();
  }
}


