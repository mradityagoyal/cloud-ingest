import { Component, Input } from '@angular/core';

import { ResourceStatus } from '../../proto/tasks.js';

@Component({
  selector: 'app-infrastructure-status-item',
  templateUrl: './infrastructure-status-item.component.html'
})
export class InfrastructureStatusItemComponent {
  @Input() public itemName: string;
  @Input() public itemStatus: string;

  // Needed to export this constant and use it in the template.
  ResourceStatus = ResourceStatus;
}
