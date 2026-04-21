import { Component, EventEmitter, Input, Output } from '@angular/core';
import { NotificationItem } from '../../global/services/notify/notify.model';

@Component({
  selector: 'app-notification-toast',
  standalone: true,
  imports: [],
  template: `
    <div class="toast-card" [class.toast-error]="item.kind === 'error'">
      <div class="toast-accent-bar"></div>

      <div class="toast-inner">
        <span class="material-symbols-outlined toast-icon">
          {{ item.kind === 'error' ? 'error' : 'info' }}
        </span>

        <div class="toast-body">
          @if (item.title) {
            <p class="toast-title">{{ item.title }}</p>
          }
          <p class="toast-message">{{ item.message }}</p>
        </div>

        <div class="toast-actions">
          @if (item.kind === 'error' && item.rawError) {
            <button class="icon-btn" (click)="detail.emit(item)" aria-label="Show details">
              <span class="material-symbols-outlined">open_in_new</span>
            </button>
          }
          <button class="icon-btn" (click)="dismiss.emit(item.id)" aria-label="Dismiss">
            <span class="material-symbols-outlined">close</span>
          </button>
        </div>
      </div>
    </div>
  `,
  styles: [`
    :host {
      display: block;
      animation: slideIn 0.25s ease-out forwards;
    }

    @keyframes slideIn {
      from { transform: translateX(110%); opacity: 0; }
      to   { transform: translateX(0);    opacity: 1; }
    }

    .toast-card {
      display: flex;
      align-items: stretch;
      border-radius: var(--launch-radius-sm);
      background: var(--launch-surface-card);
      border: 1px solid var(--launch-border-color);
      box-shadow: var(--launch-shadow-lg);
      overflow: hidden;
      min-width: 300px;
      max-width: 420px;
    }

    .toast-accent-bar {
      width: 4px;
      flex-shrink: 0;
      background: var(--launch-info-alt);
    }

    .toast-card.toast-error .toast-accent-bar {
      background: var(--launch-error-light);
    }

    .toast-inner {
      display: flex;
      align-items: center;
      gap: 10px;
      padding: 11px 8px 11px 12px;
      flex: 1;
      min-width: 0;
    }

    .toast-icon {
      font-size: 18px;
      line-height: 1;
      flex-shrink: 0;
      color: var(--launch-info-alt);
    }

    .toast-card.toast-error .toast-icon {
      color: var(--launch-error-light);
    }

    .toast-body {
      flex: 1;
      min-width: 0;
    }

    .toast-title {
      margin: 0 0 2px;
      font-size: 0.8125rem;
      font-weight: 600;
      color: var(--launch-text-primary);
    }

    .toast-message {
      margin: 0;
      font-size: 0.8125rem;
      color: var(--launch-text-secondary);
      line-height: 1.45;
      word-break: break-word;
    }

    .toast-actions {
      display: flex;
      align-items: center;
      flex-shrink: 0;
      gap: 2px;
      margin-left: 4px;
    }

    .icon-btn {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 26px;
      height: 26px;
      padding: 0;
      border: none;
      background: transparent;
      border-radius: var(--launch-radius-xs);
      cursor: pointer;
      color: var(--launch-text-muted);
      transition: background 120ms ease, color 120ms ease;
      flex-shrink: 0;

      span {
        font-size: 15px;
        line-height: 1;
      }

      &:hover {
        background: var(--launch-neutral-bg);
        color: var(--launch-text-primary);
      }
    }
  `],
})
export class NotificationToastComponent {
  @Input({ required: true }) item!: NotificationItem;
  @Output() dismiss = new EventEmitter<string>();
  @Output() detail = new EventEmitter<NotificationItem>();
}


