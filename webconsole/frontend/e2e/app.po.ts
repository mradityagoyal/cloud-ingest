import { browser, by, element } from 'protractor';

export class WebconsoleFrontEndPage {
  navigateTo() {
    return browser.get('/');
  }

  getDisplayedTitleText() {
    return element(by.css('app-root h1')).getText();
  }
}
