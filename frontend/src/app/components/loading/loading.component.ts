import { GlobalPositionStrategy, Overlay, OverlayRef } from '@angular/cdk/overlay';
import { ComponentPortal } from '@angular/cdk/portal';
import { Component, effect, OnDestroy } from '@angular/core';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { NotifyService } from '../../global/services/notify/notify.service';

@Component({
  selector: 'app-loading',
  template: '',
  standalone: true
})
export class LoadingComponent implements OnDestroy {
  private reference: OverlayRef;

  constructor(
    private overlay: Overlay,
    private notifyService: NotifyService
  ) {
    // Create the overlay first so it is ready when the effect fires
    const positionStrategy: GlobalPositionStrategy = this.overlay
      .position()
      .global()
      .bottom('0px')
      .left('0px')
      .right('0px');

    this.reference = this.overlay.create({
      positionStrategy,
      hasBackdrop: false,
      panelClass: 'loading-overlay-panel',
    });

    effect(() => {
      this.notifyService.isLoading() ? this.show() : this.hide();
    });
  }

  ngOnDestroy() {
    this.reference.dispose();
  }

  private show(): void {
    if (!this.reference.hasAttached()) {
      this.reference.attach(new ComponentPortal(LoadingProgressComponent));
    }
  }

  private hide(): void {
    if (this.reference.hasAttached()) {
      this.reference.detach();
    }
  }
}

@Component({
  selector: 'app-loading-progress',
  template: `<mat-progress-bar mode="indeterminate"></mat-progress-bar>`,
  styles: [`
    :host {
      display: block;
      width: 100%;
    }

    mat-progress-bar {
      display: block;
      width: 100%;
      /* Override MDC token for a slim 3px bar */
      --mdc-linear-progress-track-height: 3px;
      --mdc-linear-progress-active-indicator-height: 3px;
    }
  `],
  standalone: true,
  imports: [MatProgressBarModule]
})
export class LoadingProgressComponent { }
