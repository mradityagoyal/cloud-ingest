import { AuthService } from './auth/auth.service';
import { LogsComponent } from './logs/logs.component';
import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { AppComponent } from './app.component';

const appRoutes: Routes = [
  {
    path: 'logs',
    component: LogsComponent,
    canActivate: [AuthService],
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
