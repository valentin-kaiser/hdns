import { Component, effect, inject, OnDestroy, ViewChild } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { MatToolbarModule } from '@angular/material/toolbar';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { MatTooltipModule } from '@angular/material/tooltip';
import { CommonModule } from '@angular/common';
import { ApiService } from './global/services/api/api.service';
import { NotifyService } from './global/services/notify/notify.service';
import { LoadingComponent } from './components/loading/loading.component';
import { DrawerComponent } from './components/drawer/drawer.component';
import { ConfigDrawerComponent } from './pages/home/drawers/config/config-drawer.component';

@Component({
  selector: 'app-root',
  imports: [
    RouterOutlet,
    MatToolbarModule,
    MatIconModule,
    MatButtonModule,
    MatTooltipModule,
    CommonModule,
    LoadingComponent,
    DrawerComponent,
    ConfigDrawerComponent,
  ],
  templateUrl: './app.html',
  styleUrl: './app.css',
})
export class App implements OnDestroy {
  @ViewChild('configDrawer') configDrawer!: DrawerComponent;

  private readonly addressStream = inject(ApiService).streamAddress();
  readonly currentAddress = this.addressStream.latestMessage;

  constructor() {
    effect(() => {
      if (this.addressStream.isConnected()) {
        this.addressStream.send({});
      }
    });
  }

  ngOnDestroy(): void {
    this.addressStream.close();
  }

  openConfig(): void {
    this.configDrawer.open();
  }
}
