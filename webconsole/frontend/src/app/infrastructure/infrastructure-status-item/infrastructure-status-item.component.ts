import { Component, Input } from '@angular/core';
import { InfrastructureService, INFRA_STATUS } from '../infrastructure.service';

@Component({
  selector: 'app-infrastructure-status-item',
  templateUrl: './infrastructure-status-item.component.html'
})
export class InfrastructureStatusItemComponent {
  @Input() public itemName: string;
  @Input() public itemStatus: string;
  INFRA_STATUS = INFRA_STATUS;
}
