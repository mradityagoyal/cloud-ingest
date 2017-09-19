import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';

import { AngularMaterialImporterModule } from '../angular-material-importer.module';
import { JobsService } from '../jobs.service';
import { JobRunListComponent } from './job-run-list/job-run-list.component';
import { CreateRunComponent } from './create-run/create-run.component';
import { JobStatusPipe } from './job-status.pipe';

import { JobRunsRoutingModule } from './job-runs-routing.module';
import { JobRunDetailsComponent } from './job-run-details/job-run-details.component';


@NgModule({
  imports: [
    CommonModule,
    FormsModule,
    BrowserModule,
    BrowserAnimationsModule,
    AngularMaterialImporterModule,
    JobRunsRoutingModule
  ],
  declarations: [
    JobRunListComponent,
    CreateRunComponent,
    JobStatusPipe,
    JobRunDetailsComponent
  ],
  providers: [ JobsService ]
})
export class JobRunsModule { }
