import { Injectable, signal } from '@angular/core';
import { MatDialog } from '@angular/material/dialog';
import { ErrorComponent } from '../../../components/error/error.component';
import { WarningDialogComponent } from '../../../components/warning-dialog/warning-dialog.component';
import { LoggerService } from '../logger/logger.service';
import { NotificationItem, Warning } from './notify.model';

@Injectable({
  providedIn: 'root',
})
export class NotifyService {
  readonly logType = '[Service]';
  readonly logName = '[Notify]';

  /** Active notifications rendered by NotificationContainerComponent */
  readonly notifications = signal<NotificationItem[]>([]);

  /** Current loading state. */
  readonly isLoading = signal<boolean>(false);

  constructor(
    private readonly dialog: MatDialog,
    private readonly logger: LoggerService,
  ) {
    this.logger.info(`${this.logType} ${this.logName} constructor`);
  }

  // ─── Notifications ─────────────────────────────────────────────────────────

  /**
   * Show an informational toast to the user.
   */
  message(msg: string, title?: string, duration = 5000): void {
    this.addNotification({ kind: 'info', message: msg, title }, duration);
  }

  /**
   * Show an error toast to the user. The full raw error string is accessible
   * in a detail modal via the toast's info button.
   */
  error(msg: string, title?: string, duration = 5000): void {
    const displayMessage = this.extractErrorMessage(msg);
    console.log('ERROR', msg, '=>', displayMessage);
    this.addNotification(
      { kind: 'error', message: displayMessage, title, rawError: msg },
      duration,
    );
  }

  /** Dismiss a single notification by id. */
  dismissNotification(id: string): void {
    this.notifications.update(items => items.filter(n => n.id !== id));
  }

  /** Dismiss all active notifications. */
  dismissAll(): void {
    this.notifications.set([]);
  }

  /**
   * Opens a confirmation / warning dialog with the provided config.
   */
  warning(config: Warning): void {
    this.dialog.open(WarningDialogComponent, {
      data: config,
      panelClass: 'launch-dialog',
      backdropClass: 'launch-backdrop',
    });
  }

  /**
   * Opens a modal showing the full raw error message.
   */
  presentErrorDetailModal(error: string, title?: string): void {
    this.dialog.open(ErrorComponent, {
      data: { message: error, title },
      panelClass: ['launch-dialog', 'auto-height'],
      width: '480px',
    });
  }

  /** Show the loading indicator. */
  loading(): void {
    this.isLoading.set(true);
  }

  /** Hide the loading indicator. */
  dismiss(): void {
    this.isLoading.set(false);
  }

  private addNotification(partial: Omit<NotificationItem, 'id'>, duration: number): void {
    const id = crypto.randomUUID();
    const item: NotificationItem = { id, ...partial };

    this.notifications.update(items => {
      const next = [...items, item];
      // Keep at most 5 visible; drop oldest when over the limit
      return next.length > 5 ? next.slice(-5) : next;
    });

    if (duration > 0) {
      setTimeout(() => this.dismissNotification(id), duration);
    }
  }

  extractErrorMessage(error: string): string {
    if (typeof error !== 'string') {
      return 'An unknown error occurred';
    }

    const match = error.match(/\s\|\s*([^\[]+?)(?:\[(.+)\])?\s*$/);
    if (!match) return error.trim();

    const message = match[1].trim();
    const part = match[2]?.trim();

    if (!part) return message;

    const parts = part.split('|').map(p => p.trim());
    const detail = parts.length > 1 ? parts.slice(1).join(' | ').trim() : parts[0];

    return `${message}: ${detail}`;
  }
}
