import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';

import { AngularMaterialImporterModule } from '../angular-material-importer/angular-material-importer.module';
import { JobsService } from './jobs.service';
import { JobStatusPipe } from './job-status/job-status.pipe';
import { JobsRoutingModule } from './jobs-routing.module';
import { JobRunDetailsComponent } from './job-run-details/job-run-details.component';
import { JobConfigsComponent } from './job-configs/job-configs.component';
import { JobConfigAddDialogComponent } from './job-config-add-dialog/job-config-add-dialog.component';


@NgModule({
  imports: [
    CommonModule,
    FormsModule,
    BrowserModule,
    BrowserAnimationsModule,
    AngularMaterialImporterModule,
    JobsRoutingModule
  ],
  declarations: [
    JobStatusPipe,
    JobRunDetailsComponent,
    JobConfigsComponent,
    JobConfigAddDialogComponent,
  ],
  entryComponents: [JobConfigAddDialogComponent],
  providers: [ JobsService ]
})
export class JobsModule { }
