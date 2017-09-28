import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { InfrastructureComponent } from './infrastructure.component';
import { AuthService } from '../auth/auth.service';

const infrastructureRoutes: Routes = [
  {
    path: 'infrastructure',
    component: InfrastructureComponent,
    canActivate: [AuthService],
  }
];

@NgModule({
  imports: [
    RouterModule.forChild(infrastructureRoutes)
  ],
  exports: [
    RouterModule
  ]
})
export class InfrastructureRoutingModule { }
