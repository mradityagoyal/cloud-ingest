import { Component, OnInit } from '@angular/core';
import { Router } from '@angular/router';
import { AuthService } from './auth.service';

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css']
})
export class AppComponent implements OnInit {
  authServiceInitialized: boolean;
  loggedIn: boolean;
  userName: string;

  constructor(private authService: AuthService, private router: Router) { }

  ngOnInit() {
    this.authService.init();
    this.authService.loadSignInStatus().then((loggedIn) => {
      this.authServiceInitialized = true;
      this.loggedIn = loggedIn;
      if (this.loggedIn) {
        this.userName = this.authService.getCurrentUser().Name;
      }
    }).catch((err) => {
      // TODO(b/64808404): Show the error messages in the web console.
    });
  }

  signOut() {
    this.authService.signOut().then(() => {
      this.loggedIn = false;
      this.router.navigateByUrl('/');
    }).catch((err) => {
      // TODO(b/64808404): Show the error messages in the web console.
    });
  }

  signIn() {
    this.authService.signIn().then(() => {
      this.loggedIn = true;
      this.userName = this.authService.getCurrentUser().Name;
    }).catch((err) => {
      // TODO(b/64808404): Show the error messages in the web console.
    });
  }
}
