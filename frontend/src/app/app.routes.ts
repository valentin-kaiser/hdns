import { Routes } from '@angular/router';
import { HomeComponent } from './pages/home/home.component';
import { RecordOverviewComponent } from './pages/overview/record-overview.component';

export const routes: Routes = [
  { path: '', component: HomeComponent },
  { path: 'overview/:recordId', component: RecordOverviewComponent },
  { path: '**', redirectTo: '' },
];
