import {AppComponent} from './app.component';
import {AuthService} from './auth/auth.service';
import {LogsComponent} from './logs/logs.component';
import {AgentComponent} from './agent/agent.component';
import {NgModule} from '@angular/core';
import {RouterModule, Routes } from '@angular/router';
import {BandwidthControlComponent} from './bandwidth-control/bandwidth-control.component';

const appRoutes: Routes = [
  {
    path: 'logs',
    component: LogsComponent,
    canActivate: [AuthService],
  },
  {
    path: 'agent',
    component: AgentComponent,
    canActivate: [AuthService],
  },
  {
    path: 'bandwidth-control',
    component: BandwidthControlComponent,
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
