import { Injectable } from '@angular/core';
import { DrawerComponent } from '../../../components/drawer/drawer.component';

/**
 * Service to track all open DrawerComponent instances.
 * Allows global close-all functionality and z-index management.
 */
@Injectable({ providedIn: 'root' })
export class DrawerService {
  private drawers = new Set<DrawerComponent>();

  register(drawer: DrawerComponent): void {
    this.drawers.add(drawer);
  }

  unregister(drawer: DrawerComponent): void {
    this.drawers.delete(drawer);
  }

  closeAll(): void {
    this.drawers.forEach(d => d.close());
  }
}
