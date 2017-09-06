import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { HttpClientModule, HTTP_INTERCEPTORS } from '@angular/common/http';
import { FormsModule } from '@angular/forms';
import { AngularMaterialImporterModule } from './angular-material-importer.module';

import { AppComponent } from './app.component';
import { JobsService } from './jobs.service';
import { AuthInterceptor } from './auth-interceptor';
import { AuthService } from './auth.service';
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
    BrowserAnimationsModule,
    HttpClientModule,
    FormsModule,
    AngularMaterialImporterModule,
    RouterModule.forRoot([
      {
        path: 'jobconfigs',
        component: JobConfigsComponent,
        canActivate: [AuthService],
      },
      {
        path: 'jobruns',
        component: JobRunsComponent,
        canActivate: [AuthService],
      },
      {
        path: 'createconfig',
        component: CreateConfigComponent,
        canActivate: [AuthService],
      },
      {
        path: 'createrun',
        component: CreateRunComponent,
        canActivate: [AuthService],
      }
    ])
  ],
  providers: [
    JobsService,
    AuthService,
    {
      provide: HTTP_INTERCEPTORS,
      useClass: AuthInterceptor,
      multi: true,
    },
  ],
  bootstrap: [AppComponent]
})
export class AppModule { }
