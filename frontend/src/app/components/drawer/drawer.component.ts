import {
  Overlay,
  OverlayConfig,
  OverlayModule,
  OverlayRef,
} from '@angular/cdk/overlay';
import { TemplatePortal } from '@angular/cdk/portal';
import { CommonModule } from '@angular/common';
import {
  Component,
  EventEmitter,
  HostListener,
  Input,
  OnDestroy,
  Output,
  TemplateRef,
  ViewChild,
  ViewContainerRef,
} from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { Subject } from 'rxjs';
import { takeUntil } from 'rxjs/operators';
import { DrawerService } from '../../global/services/drawer/drawer.service';

export interface DrawerBreakpoint {
  maxWidth: number;
  width: number;
}

/**
 * Reusable drawer overlay component.
 *
 * Usage:
 * ```html
 * <app-drawer [width]="50" [breakpoints]="[{ maxWidth: 768, width: 100 }]">
 *   <div header>Header content</div>
 *   <div content>Body content</div>
 *   <div footer>Footer content</div>
 * </app-drawer>
 * ```
 */
@Component({
  selector: 'app-drawer',
  templateUrl: './drawer.component.html',
  styleUrls: ['./drawer.component.scss'],
  imports: [CommonModule, OverlayModule, MatIconModule],
})
export class DrawerComponent implements OnDestroy {
  private reference: OverlayRef | null = null;
  private destroy$ = new Subject<void>();
  protected visible = false;

  @Input() side: 'left' | 'right' = 'right';
  @Input() vertical: 'top' | 'bottom' = 'top';
  @Input() width = 50;
  @Input() height = 100;
  @Input() animate = true;
  @Input() backdrop = true;
  @Input() breakpoints: DrawerBreakpoint[] = [
    { maxWidth: 1200, width: 100 },
  ];

  @Output() closed = new EventEmitter<void>();

  @ViewChild('drawer', { static: true }) drawerTemplate!: TemplateRef<any>;

  constructor(
    private overlay: Overlay,
    private viewContainerRef: ViewContainerRef,
    private drawerService: DrawerService,
  ) {
    this.drawerService.register(this);
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
    this.drawerService.unregister(this);
    this.close();
  }

  @HostListener('window:resize')
  onWindowResize(): void {
    if (this.reference) this.update();
  }

  public open(): void {
    if (this.reference) return;
    this.create();
    this.visible = true;
  }

  public close(): void {
    if (!this.reference) return;

    if (this.animate) {
      this.reference.overlayElement
        .querySelector('.panel')
        ?.classList.add('slide-out');
    }

    setTimeout(() => {
      this.reference?.dispose();
      this.reference = null;
      this.visible = false;
    }, this.animate ? 240 : 0);
  }

  public isOpen(): boolean {
    return this.visible;
  }

  public isTop(): boolean {
    const container = document.querySelector('.cdk-overlay-container');
    if (!container) return false;
    const overlays = Array.from(container.children);
    if (overlays.length === 0) return false;
    const children = Array.from(overlays[overlays.length - 1].children);
    return children.some(child => child === this.reference?.overlayElement);
  }

  private create(): void {
    const config = this.buildConfig();
    this.reference = this.overlay.create(config);

    const portal = new TemplatePortal(this.drawerTemplate, this.viewContainerRef, {
      side: this.side,
      width: this.calculateWidth(),
    });

    this.reference.attach(portal);

    this.reference
      .backdropClick()
      .pipe(takeUntil(this.destroy$))
      .subscribe(() => this.closed.emit());

    this.reference
      .keydownEvents()
      .pipe(takeUntil(this.destroy$))
      .subscribe(event => {
        if (event.key === 'Escape') this.closed.emit();
      });
  }

  private calculateWidth(): number {
    const currentWidth = window.innerWidth;
    if (this.breakpoints?.length) {
      const sorted = [...this.breakpoints].sort((a, b) => a.maxWidth - b.maxWidth);
      for (const bp of sorted) {
        if (currentWidth <= bp.maxWidth) return bp.width;
      }
    }
    return this.width;
  }

  private update(): void {
    if (!this.reference) return;

    const positionStrategy = this.overlay
      .position()
      .global()
      [this.side === 'left' ? 'start' : 'end']('0')
      [this.vertical === 'top' ? 'top' : 'bottom']('0');

    this.reference.updatePositionStrategy(positionStrategy);
    this.reference.updateSize({
      width: `${this.calculateWidth()}%`,
      height: `${this.height}%`,
    });
  }

  private buildConfig(): OverlayConfig {
    const width = `${this.calculateWidth()}%`;
    const height = `${this.height}%`;

    const positionStrategy = this.overlay
      .position()
      .global()
      [this.side === 'left' ? 'start' : 'end']('0')
      [this.vertical === 'top' ? 'top' : 'bottom']('0');

    return {
      positionStrategy,
      hasBackdrop: this.backdrop,
      backdropClass: 'cdk-overlay-dark-backdrop',
      panelClass: [
        'drawer-overlay-pane',
        `drawer-overlay-pane-${this.side}`,
        this.backdrop ? '' : 'drawer-no-backdrop',
      ],
      scrollStrategy: this.overlay.scrollStrategies.block(),
      disposeOnNavigation: true,
      width,
      height,
    };
  }
}
