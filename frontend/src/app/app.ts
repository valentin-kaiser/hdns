import { CommonModule } from '@angular/common';
import { ApplicationRef, Component, createComponent, EnvironmentInjector, inject, OnDestroy, OnInit } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressBarModule } from '@angular/material/progress-bar';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatTooltipModule } from '@angular/material/tooltip';
import { RouterOutlet } from '@angular/router';
import { Subscription } from 'rxjs';
import { NotificationContainerComponent } from './components/notification-container/notification-container.component';
import { ApiService } from './global/services/api/api.service';
import { NotifyService } from './global/services/notify/notify.service';
import { ConfigDrawerComponent } from './pages/home/drawers/config/config-drawer.component';

@Component({
  selector: 'app-root',
  imports: [
    RouterOutlet,
    MatToolbarModule,
    MatIconModule,
    MatButtonModule,
    MatTooltipModule,
    MatProgressBarModule,
    CommonModule,
    ConfigDrawerComponent,
  ],
  templateUrl: './app.html',
  styleUrl: './app.css',
})
export class App implements OnInit, OnDestroy {
  readonly isLoading = inject(NotifyService).isLoading;

  private readonly addressStream = inject(ApiService).streamAddress();
  readonly currentAddress = toSignal(this.addressStream.messages$, { initialValue: null });

  private readonly connectedSub: Subscription = this.addressStream.connect$.subscribe(() => {
    this.addressStream.send({});
  });
  
  private notificationContainerRef?: ReturnType<typeof createComponent<NotificationContainerComponent>>;

  constructor(
    private appRef: ApplicationRef,
    private injector: EnvironmentInjector,
  ) { }

  ngOnInit(): void {
    // Mount the notification container directly on document.body so it is at the
    // same DOM level as the CDK overlay container and always renders on top.
    this.notificationContainerRef = createComponent(NotificationContainerComponent, {
      environmentInjector: this.injector,
    });
    this.appRef.attachView(this.notificationContainerRef.hostView);
    document.body.appendChild(this.notificationContainerRef.location.nativeElement);
  }

  ngOnDestroy(): void {
    this.connectedSub.unsubscribe();
    this.addressStream.close();
    if (this.notificationContainerRef) {
      this.appRef.detachView(this.notificationContainerRef.hostView);
      this.notificationContainerRef.destroy();
    }
  }
}
