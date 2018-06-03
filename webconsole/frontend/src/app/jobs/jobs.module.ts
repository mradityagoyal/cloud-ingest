import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';

import { AngularMaterialImporterModule } from '../angular-material-importer/angular-material-importer.module';
import { JobConfigAddDialogComponent } from './job-config-add-dialog/job-config-add-dialog.component';
import { JobConfigsComponent } from './job-configs/job-configs.component';
import { ENABLE_POLLING, JobRunDetailsComponent } from './job-run-details/job-run-details.component';
import { JobsRoutingModule } from './jobs-routing.module';
import { JobsService } from './jobs.service';


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
    JobRunDetailsComponent,
    JobConfigsComponent,
    JobConfigAddDialogComponent,
  ],
  entryComponents: [JobConfigAddDialogComponent],
  providers: [ {provide: JobsService, useClass: JobsService},
                {provide: ENABLE_POLLING, useValue: true} ]
})
export class JobsModule { }
