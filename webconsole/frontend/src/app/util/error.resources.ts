import { HttpErrorResponse } from '@angular/common/http';

export class HttpErrorResponseFormatter {
  static getTitle(errorResponse: HttpErrorResponse) {
    if (errorResponse.error != null && errorResponse.error.error != null && typeof errorResponse.error.error === 'string') {
      return errorResponse.error.error;
    } else {
      return errorResponse.statusText;
    }
  }

  static getMessage(errorResponse: HttpErrorResponse) {
    let errorMessage = '';
    if (errorResponse.error != null && errorResponse.error.message != null && typeof errorResponse.error.message === 'string') {
      errorMessage = errorResponse.error.message;
      if (errorResponse.error.traceback != null && typeof errorResponse.error.traceback === 'string') {
        errorMessage  = errorMessage + '\n' + errorResponse.error.traceback;
      }
    } else {
      errorMessage = errorResponse.message;
    }
    return errorMessage;
  }
}
