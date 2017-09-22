import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { AuthService } from '../auth.service';
import { JobRunDetailsComponent } from './job-run-details/job-run-details.component';

const jobRunsRoutes: Routes = [
  {
    path: 'jobconfigs/:configId/:runId',
    component: JobRunDetailsComponent,
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
