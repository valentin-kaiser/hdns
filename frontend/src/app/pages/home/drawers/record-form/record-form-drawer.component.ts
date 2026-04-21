import { CommonModule } from '@angular/common';
import {
  ChangeDetectorRef,
  Component,
  EventEmitter,
  inject,
  OnChanges,
  OnDestroy,
  Output,
  ViewChild,
} from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatStepper, MatStepperModule } from '@angular/material/stepper';
import { debounceTime, distinctUntilChanged, Subscription } from 'rxjs';
import { DrawerComponent } from '../../../../components/drawer/drawer.component';
import { Record as DnsRecord, Zone } from '../../../../global/model/api';
import { ApiService } from '../../../../global/services/api/api.service';
import { NotifyService } from '../../../../global/services/notify/notify.service';

@Component({
  selector: 'app-record-form-drawer',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    MatFormFieldModule,
    MatInputModule,
    MatSelectModule,
    MatButtonModule,
    MatIconModule,
    MatStepperModule,
    DrawerComponent,
  ],
  template: `
    <app-drawer #drawer [width]="45" [breakpoints]="[{ maxWidth: 768, width: 100 }]">
      <div class="drawer-header">
        <h3 class="drawer-title">{{ record ? 'Edit Record' : 'Add Record' }}</h3>
      </div>

      <div class="drawer-body">
        <mat-stepper #stepper orientation="vertical" [linear]="!record">
          <!-- Step 1: Token -->
          <mat-step [stepControl]="tokenGroup" label="API Token">
            <form [formGroup]="tokenGroup" class="step-form">
              <p class="step-hint">Enter your Hetzner DNS API token to load available zones.</p>
              <mat-form-field appearance="outline" class="full-width">
                <mat-label>Hetzner API Token</mat-label>
                <input matInput type="password" formControlName="token" placeholder="Enter token" />
                <mat-icon matSuffix>key</mat-icon>
              </mat-form-field>
            </form>
          </mat-step>

          <!-- Step 2: Zone -->
          <mat-step [stepControl]="zoneGroup" label="Zone">
            <form [formGroup]="zoneGroup" class="step-form">
              <p class="step-hint">Select the DNS zone this record belongs to.</p>
              <mat-form-field appearance="outline" class="full-width">
                <mat-label>Zone</mat-label>
                <mat-select formControlName="zoneId">
                  @for (z of zones; track z.id) {
                    <mat-option [value]="z.id">{{ z.name }}</mat-option>
                  }
                </mat-select>
                @if (zonesLoading) {
                  <mat-hint>Loading zones...</mat-hint>
                }
                @if (zones.length === 0) {
                  <mat-hint>No zones found for this token.</mat-hint>
                }
              </mat-form-field>
            </form>
          </mat-step>

          <!-- Step 3: Details -->
          <mat-step [stepControl]="detailsGroup" label="Record Details">
            <form [formGroup]="detailsGroup" class="step-form">
              <mat-form-field appearance="outline" class="full-width">
                <mat-label>Record name</mat-label>
                <input matInput formControlName="name" placeholder="e.g. @ or subdomain" />
              </mat-form-field>
              <mat-form-field appearance="outline" class="full-width">
                <mat-label>TTL (seconds)</mat-label>
                <input matInput type="number" formControlName="ttl" min="1" />
              </mat-form-field>
            </form>
          </mat-step>
        </mat-stepper>
      </div>

      <div class="drawer-footer">
        <button mat-stroked-button type="button" (click)="drawer.close()">Cancel</button>
        <button
          mat-flat-button
          color="primary"
          type="button"
          [disabled]="tokenGroup.invalid || zoneGroup.invalid || detailsGroup.invalid || saving"
          (click)="save()"
        >
          {{ saving ? 'Saving...' : 'Save' }}
        </button>
      </div>
    </app-drawer>
  `,
  styles: [
    `
      :host {
        display: flex;
        flex: 1;
        min-height: 0;
        flex-direction: column;
        height: 100%;
      }
      .drawer-header {
        flex-shrink: 0;
        padding: 20px 24px 16px;
        border-bottom: 1px solid var(--launch-border-color);
      }
      .drawer-title {
        margin: 0;
        font-size: 1rem;
        font-weight: 600;
        color: var(--launch-text-primary);
      }
      .drawer-body {
        flex: 1;
        min-height: 0;
        overflow-y: auto;
        padding: 8px 16px 16px;
      }
      .drawer-footer {
        display: flex;
        justify-content: flex-end;
        gap: 8px;
        padding: 16px 24px 20px;
        border-top: 1px solid var(--launch-border-color);
        flex-shrink: 0;
      }
      .step-hint {
        margin: 0 0 16px;
        font-size: 0.875rem;
        color: var(--launch-text-muted);
      }
      .step-form {
        display: flex;
        flex-direction: column;
        gap: 4px;
        padding-top: 12px;
      }
      .full-width {
        width: 100%;
      }
    `,
  ],
})
export class RecordFormDrawerComponent implements OnChanges, OnDestroy {
  record: DnsRecord | null = null;
  @ViewChild('stepper') stepper!: MatStepper;
  @ViewChild('drawer') drawer!: DrawerComponent;

  @Output() onSave = new EventEmitter<void>();

  zones: Zone[] = [];
  zonesLoading = false;
  saving = false;
  private readonly autoAdvanceDelayMs = 350;
  private autoAdvanceSuspended = false;
  private tokenAdvanceTimer: ReturnType<typeof setTimeout> | null = null;
  private zoneAdvanceTimer: ReturnType<typeof setTimeout> | null = null;

  private readonly fb = inject(FormBuilder);
  private readonly api = inject(ApiService);
  private readonly notify = inject(NotifyService);
  private readonly cdr = inject(ChangeDetectorRef);
  private readonly subscriptions = new Subscription();

  tokenGroup = this.fb.group({
    token: ['', Validators.required],
  });

  zoneGroup = this.fb.group({
    zoneId: [0, [Validators.required, Validators.min(1)]],
  });

  detailsGroup = this.fb.group({
    name: ['', Validators.required],
    ttl: [60, [Validators.required, Validators.min(1)]],
  });

  constructor() {
    this.subscriptions.add(
      this.tokenGroup
        .get('token')!
        .valueChanges.pipe(debounceTime(600), distinctUntilChanged())
        .subscribe((token) => {
          if (token) {
            this.loadZones(token);
          } else {
            this.clearAdvanceTimer('token');
            this.zones = [];
            this.zoneGroup.patchValue({ zoneId: 0 });
          }
        }),
    );

    this.subscriptions.add(
      this.tokenGroup.statusChanges.pipe(distinctUntilChanged()).subscribe(() => {
        this.scheduleStepAdvance('token');
      }),
    );

    this.subscriptions.add(
      this.zoneGroup.statusChanges.pipe(distinctUntilChanged()).subscribe(() => {
        this.scheduleStepAdvance('zone');
      }),
    );
  }

  ngOnChanges(): void {
    this.autoAdvanceSuspended = true;
    this.clearAdvanceTimers();

    if (this.record) {
      this.tokenGroup.patchValue({ token: this.record.token });
      this.zoneGroup.patchValue({ zoneId: this.record.zoneId });
      this.detailsGroup.patchValue({ name: this.record.name, ttl: this.record.ttl });
      const token = this.record.token;
      setTimeout(() => {
        if (token) this.loadZones(token);
        if (this.stepper) this.stepper.selectedIndex = 2;
        this.autoAdvanceSuspended = false;
      });
    } else {
      this.tokenGroup.reset({ token: '' });
      this.zoneGroup.reset({ zoneId: 0 });
      this.detailsGroup.reset({ name: '', ttl: 60 });
      this.zones = [];
      setTimeout(() => {
        if (this.stepper) this.stepper.selectedIndex = 0;
        this.autoAdvanceSuspended = false;
      });
    }
  }

  ngOnDestroy(): void {
    this.clearAdvanceTimers();
    this.subscriptions.unsubscribe();
  }

  loadZones(token: string): void {
    this.zonesLoading = true;
    this.cdr.markForCheck();
    this.api.getZones({ token }).subscribe({
      next: (res) => {
        this.zones = res.zones ?? [];
        if (!this.zones.some((zone) => zone.id === this.zoneGroup.value.zoneId)) {
          this.zoneGroup.patchValue({ zoneId: 0 });
        }
        this.zonesLoading = false;
        this.cdr.markForCheck();
      },
      error: (err) => {
        this.zones = [];
        this.zoneGroup.patchValue({ zoneId: 0 });
        this.zonesLoading = false;
        this.cdr.markForCheck();
        this.notify.error(err?.error, 'Failed to load zones');
      },
    });
  }

  private scheduleStepAdvance(step: 'token' | 'zone'): void {
    if (this.autoAdvanceSuspended || !this.stepper) return;

    const group = step === 'token' ? this.tokenGroup : this.zoneGroup;
    const expectedIndex = step === 'token' ? 0 : 1;

    this.clearAdvanceTimer(step);

    if (this.stepper.selectedIndex !== expectedIndex || group.invalid) return;

    const timer = setTimeout(() => {
      if (
        this.autoAdvanceSuspended ||
        !this.stepper ||
        this.stepper.selectedIndex !== expectedIndex ||
        group.invalid
      ) {
        return;
      }

      this.stepper.next();
    }, this.autoAdvanceDelayMs);

    if (step === 'token') {
      this.tokenAdvanceTimer = timer;
    } else {
      this.zoneAdvanceTimer = timer;
    }
  }

  private clearAdvanceTimer(step: 'token' | 'zone'): void {
    const timer = step === 'token' ? this.tokenAdvanceTimer : this.zoneAdvanceTimer;
    if (!timer) return;

    clearTimeout(timer);

    if (step === 'token') {
      this.tokenAdvanceTimer = null;
    } else {
      this.zoneAdvanceTimer = null;
    }
  }

  private clearAdvanceTimers(): void {
    this.clearAdvanceTimer('token');
    this.clearAdvanceTimer('zone');
  }

  open() {
    this.drawer.open();
  }

  save(): void {
    if (this.tokenGroup.invalid || this.zoneGroup.invalid || this.detailsGroup.invalid) return;
    const token = this.tokenGroup.value.token!;
    const zoneId = this.zoneGroup.value.zoneId!;
    const name = this.detailsGroup.value.name!;
    const ttl = this.detailsGroup.value.ttl!;
    const domain =
      this.zones.find((zone) => zone.id === zoneId)?.name?.trim() ??
      this.record?.domain?.trim() ??
      '';
    if (!domain) {
      this.notify.error('Select a valid zone before saving.', 'Save failed');
      return;
    }
    const payload: DnsRecord = {
      ...(this.record ?? {
        id: 0,
        createdAt: 0,
        updatedAt: 0,
        addressId: 0,
        domain: '',
        address: undefined,
        lastRefresh: 0,
      }),
      token,
      zoneId,
      domain,
      name,
      ttl,
    };
    this.saving = true;
    this.api.upsertRecord(payload).subscribe({
      next: () => {
        this.onSave.emit();
        this.saving = false;
        this.notify.message('Record saved.');
        this.drawer.close();
      },
      error: (err) => {
        this.saving = false;
        this.notify.error(err?.error, 'Save failed');
      },
    });
  }
}
