import { Component, inject, Input, OnInit } from '@angular/core';
import { MatButtonModule } from '@angular/material/button';
import { MAT_DIALOG_DATA, MatDialogModule } from '@angular/material/dialog';

@Component({
  selector: 'app-error',
  template: `
    @if (message) {
      <div class="dialog-header">
        <span class="material-symbols-outlined header-icon">error</span>
        <span class="header-title">{{ title ?? 'Error' }}</span>
      </div>

      <mat-dialog-content class="dialog-body">
        <p>{{ message }}</p>
      </mat-dialog-content>

      <mat-dialog-actions align="end" class="dialog-footer">
        <button mat-button mat-dialog-close class="close-btn">Close</button>
      </mat-dialog-actions>
    }
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
      color: var(--launch-error-light);
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
        font-size: 0.8125rem;
        color: var(--launch-text-secondary);
        line-height: 1.65;
        word-break: break-word;
        white-space: pre-wrap;
      }
    }

    .dialog-footer {
      padding: 8px 16px 12px !important;
      border-top: 1px solid var(--launch-border-color);
      gap: 8px;
      margin: 0 !important;
    }

    .close-btn {
      color: var(--launch-text-secondary) !important;
    }
  `],
  imports: [MatDialogModule, MatButtonModule],
})
export class ErrorComponent implements OnInit {
  @Input() title?: string;
  @Input() message?: string;

  private readonly dialogData = inject(MAT_DIALOG_DATA, { optional: true }) as
    | { message?: string; title?: string }
    | null;

  ngOnInit(): void {
    if (this.dialogData) {
      this.message ??= this.dialogData.message;
      this.title ??= this.dialogData.title;
    }
  }
}
