import { Component, OnInit } from '@angular/core';
import { AddressCardComponent } from './sections/address-card/address-card.component';
import { RecordsTableComponent } from './sections/records-table/records-table.component';

@Component({
  selector: 'app-home',
  standalone: true,
  imports: [AddressCardComponent, RecordsTableComponent],
  template: `
    <div class="page-container">
      <app-address-card />
      <app-records-table />
    </div>
  `,
})
export class HomeComponent {}
