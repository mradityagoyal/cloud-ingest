/**
  * AuthInterceptor intercepts http requests and adds the current Google
  * signed-in user authorization header to the intercepted http requests.
  */
import { Injectable } from '@angular/core';
import { HttpEvent, HttpInterceptor, HttpHandler, HttpRequest } from '@angular/common/http';
import { Observable } from 'rxjs';
import { AuthService } from './auth.service';

@Injectable()
export class AuthInterceptor implements HttpInterceptor {
  constructor(private auth: AuthService) {}

  intercept(req: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
    // Get the auth header from the service.
    const authHeader = this.auth.getAuthorizationHeader();
    // Clone the request to add the new header.
    const authReq = req.clone({setHeaders: {'Authorization': authHeader}});
    // Pass on the cloned request instead of the original request.
    return next.handle(authReq);
  }
}

