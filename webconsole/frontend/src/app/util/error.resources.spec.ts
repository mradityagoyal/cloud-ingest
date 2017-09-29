import { TestBed, async } from '@angular/core/testing';
import { HttpErrorResponse, HttpHeaders, HttpEventType } from '@angular/common/http';
import { HttpErrorResponseFormatter } from './error.resources';

const FAKE_HTTP_ERROR1: HttpErrorResponse = {
  name: 'HttpErrorResponse',
  ok: false,
  status: 400,
  headers: new HttpHeaders(),
  type: HttpEventType.Response,
  url: '/fake/url',
  message: 'fake http response 1',
  error: {
    error: 'fake error title',
    message: 'fake error message'
  },
  statusText: 'fake error status'
};

const FAKE_HTTP_ERROR2: HttpErrorResponse = {
  name: 'HttpErrorResponse',
  ok: false,
  status: 400,
  headers: new HttpHeaders(),
  type: HttpEventType.Response,
  url: '/fake/url',
  message: 'fake http response 2',
  error: {
    error: {fakeField : 'unexpected format'},
    message: {fakeField2: 'unexpected message format'}
  },
  statusText: 'fake error status'
};

const FAKE_HTTP_ERROR3: HttpErrorResponse = {
  name: 'HttpErrorResponse',
  ok: false,
  status: 400,
  headers: new HttpHeaders(),
  type: HttpEventType.Response,
  url: '/fake/url',
  message: 'fake http response 3',
  error: {
    error: 'fake error title',
    message: 'fake error message',
    traceback: 'fake traceback'
  },
  statusText: 'fake error status'
};

describe('HttpErrorResponseFormatter', () => {

  it('getTitle should return the error.error field', () => {
    expect(HttpErrorResponseFormatter.getTitle(FAKE_HTTP_ERROR1)).toBe('fake error title');
  });

  it('getTitle should return the statusText when the error.error field is not a string', () => {
    expect(HttpErrorResponseFormatter.getTitle(FAKE_HTTP_ERROR2)).toBe('fake error status');
  });

  it('getMessage should return the error.message field', () => {
    expect(HttpErrorResponseFormatter.getMessage(FAKE_HTTP_ERROR1)).toBe('fake error message');
  });

  it('getMessage should return the message property when error.message is not a string', () => {
    expect(HttpErrorResponseFormatter.getMessage(FAKE_HTTP_ERROR2)).toBe('fake http response 2');
  });

  it('getMessage should return the error.message and the error.traceback field', () => {
    expect(HttpErrorResponseFormatter.getMessage(FAKE_HTTP_ERROR3)).toContain('fake error message');
    expect(HttpErrorResponseFormatter.getMessage(FAKE_HTTP_ERROR3)).toContain('fake traceback');
  });

});
