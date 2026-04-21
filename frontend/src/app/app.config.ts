import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http';
import { ApplicationConfig, provideBrowserGlobalErrorListeners } from '@angular/core';
import { provideNativeDateAdapter } from '@angular/material/core';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { provideRouter } from '@angular/router';

import { OVERLAY_DEFAULT_CONFIG } from '@angular/cdk/overlay';
import { MAT_ICON_DEFAULT_OPTIONS } from '@angular/material/icon';
import { routes } from './app.routes';
import { ConsoleLoggerService } from './global/services/logger/console-logger.service';
import { LoggerService } from './global/services/logger/logger.service';

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideRouter(routes),
    provideHttpClient(withInterceptorsFromDi()),
    provideAnimationsAsync(),
    provideNativeDateAdapter(),
    { provide: LoggerService, useClass: ConsoleLoggerService },
    { provide: MAT_ICON_DEFAULT_OPTIONS, useValue: { fontSet: 'material-symbols-outlined' } },
    // Disable CDK Popover API so all overlays use regular z-index stacking.
    // This ensures the notification container (z-index: 9999) always renders
    // above CDK overlays (z-index: 1000) without top-layer conflicts.
    { provide: OVERLAY_DEFAULT_CONFIG, useValue: { usePopover: false } },
  ],
};
