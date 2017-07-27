import { BrowserModule } from '@angular/platform-browser';
import { NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { HttpClientModule } from '@angular/common/http';
import { FormsModule } from '@angular/forms';

import { AppComponent } from './app.component';
import { JobsService } from './jobs.service';
import { JobConfigsComponent } from './job-configs.component';
import { JobRunsComponent } from './job-runs.component';
import { CreateConfigComponent } from './create-config.component';
import { CreateRunComponent } from './create-run.component';

@NgModule({
  declarations: [
    AppComponent, JobConfigsComponent, JobRunsComponent, CreateConfigComponent,
    CreateRunComponent
  ],
  imports: [
    BrowserModule,
    HttpClientModule,
    FormsModule,
    RouterModule.forRoot([
      {
        path: 'jobconfigs',
        component: JobConfigsComponent
      },
      {
        path: 'jobruns',
        component: JobRunsComponent
      },
      {
        path: 'createconfig',
        component: CreateConfigComponent
      },
      {
        path: 'createrun',
        component: CreateRunComponent
      }
    ])
  ],
  providers: [JobsService],
  bootstrap: [AppComponent]
})
export class AppModule { }
