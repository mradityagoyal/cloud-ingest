import { Component, Input } from '@angular/core';

import { INFRA_STATUS } from '../infrastructure.resources';

@Component({
  selector: 'app-infrastructure-status-item',
  templateUrl: './infrastructure-status-item.component.html'
})
export class InfrastructureStatusItemComponent {
  @Input() public itemName: string;
  @Input() public itemStatus: string;
  INFRA_STATUS = INFRA_STATUS;
}
