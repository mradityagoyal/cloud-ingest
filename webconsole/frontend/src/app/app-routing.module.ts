import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { AuthService } from './auth.service';
import { JobConfigsComponent } from './job-configs.component';
import { JobRunListComponent } from './job-runs/job-run-list/job-run-list.component';
import { CreateRunComponent } from './job-runs/create-run/create-run.component';
import { JobConfigAddDialogComponent } from './job-config-add-dialog.component';
import { InfrastructureComponent } from './infrastructure.component';

const appRoutes: Routes = [
  {
    path: 'jobconfigs',
    component: JobConfigsComponent,
    canActivate: [AuthService],
  },
  {
    path: 'infrastructure',
    component: InfrastructureComponent,
    canActivate: [AuthService]
  }
];

@NgModule({
  imports: [
    RouterModule.forRoot(
      appRoutes
    )
  ],
  exports: [
    RouterModule
  ]
})
export class AppRoutingModule { }
