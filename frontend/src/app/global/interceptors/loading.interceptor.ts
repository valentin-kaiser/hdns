import { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { finalize } from 'rxjs';
import { NotifyService } from '../services/notify/notify.service';

export const loadingInterceptor: HttpInterceptorFn = (req, next) => {
  const notify = inject(NotifyService);
  notify.loading();
  return next(req).pipe(finalize(() => notify.dismiss()));
};
