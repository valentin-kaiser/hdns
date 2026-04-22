import { Component, inject } from '@angular/core';
import { NotificationItem } from '../../global/services/notify/notify.model';
import { NotifyService } from '../../global/services/notify/notify.service';
import { NotificationToastComponent } from '../notification-toast/notification-toast.component';

@Component({
  selector: 'app-notification-container',
  standalone: true,
  imports: [NotificationToastComponent],
  template: `
    <div class="notification-stack">
      @for (item of notify.notifications(); track item.id) {
        <app-notification-toast
          [item]="item"
          (dismiss)="notify.dismissNotification($event)"
          (detail)="showDetail($event)">
        </app-notification-toast>
      }

      @if (notify.notifications().length > 1) {
        <button class="dismiss-all-btn" (click)="notify.dismissAll()">
          Dismiss all {{ notify.notifications().length }} notifications
        </button>
      }
    </div>
  `,
  styles: [`
    :host {
      position: fixed;
      bottom: 24px;
      right: 24px;
      z-index: 9999;
      pointer-events: none;
    }

    .notification-stack {
      display: flex;
      flex-direction: column-reverse;
      gap: 8px;
      align-items: flex-end;
      pointer-events: all;
    }

    .dismiss-all-btn {
      align-self: flex-end;
      background: none;
      border: none;
      font-size: 0.75rem;
      color: var(--launch-text-secondary, #94a3b8);
      cursor: pointer;
      padding: 2px 4px;
      border-radius: 4px;
      text-decoration: underline;

      &:hover {
        color: var(--launch-text-primary, #f1f5f9);
      }
    }
  `],
})
export class NotificationContainerComponent {
  readonly notify = inject(NotifyService);

  showDetail(item: NotificationItem): void {
    if (item.rawError) {
      this.notify.presentErrorDetailModal(item.rawError, item.title);
    }
  }
}
