import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { AuthService } from '../auth.service';
import { JobRunListComponent } from './job-run-list/job-run-list.component';
import { CreateRunComponent } from './create-run/create-run.component';
import { JobRunDetailsComponent } from './job-run-details/job-run-details.component';

const jobRunsRoutes: Routes = [
  {
    path: 'jobruns',
    component: JobRunListComponent,
    canActivate: [AuthService],
  },
  {
    path: 'jobruns/:configId/:runId',
    component: JobRunDetailsComponent,
    canActivate: [AuthService],
  },
  {
    path: 'createrun',
    component: CreateRunComponent,
    canActivate: [AuthService],
  }
];

@NgModule({
  imports: [
    RouterModule.forChild(jobRunsRoutes)
  ],
  exports: [
    RouterModule
  ]
})
export class JobRunsRoutingModule { }
