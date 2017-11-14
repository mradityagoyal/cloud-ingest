import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { AuthService } from '../auth/auth.service';
import { JobRunDetailsComponent } from './job-run-details/job-run-details.component';
import { JobConfigsComponent } from './job-configs/job-configs.component';
import { JobTasksComponent } from './job-tasks/job-tasks.component';

const jobsRoutes: Routes = [
  {
    path: 'jobs',
    component: JobConfigsComponent,
    canActivate: [AuthService],
  },
  {
    path: 'jobs/:configId/:runId',
    component: JobRunDetailsComponent,
    canActivate: [AuthService],
  },
  {
    path: 'jobs/:configId/:runId/tasks',
    component: JobTasksComponent,
    canActivate: [AuthService],
  }
];

@NgModule({
  imports: [
    RouterModule.forChild(jobsRoutes)
  ],
  exports: [
    RouterModule
  ]
})
export class JobsRoutingModule { }
