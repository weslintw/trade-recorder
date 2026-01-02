/**
 * UI Definitions for Trade Recorder
 * This file contains selectors and expected text for E2E testing.
 */

export const UI_DEFINITIONS = {
  HOME: {
    URL: '/',
    BRAND: {
      SELECTOR: 'a.nav-brand',
      TEXT: 'v1.0.0'
    },
    DASHBOARD_LINK: {
      SELECTOR: 'a.dashboard-link',
      TEXT: '📊 統計面板'
    },
    ACCOUNTS_LINK: {
      SELECTOR: "a.nav-icon-btn[href='/accounts']",
      TITLE: '帳號管理'
    },
    ADD_PLAN_BTN: {
      SELECTOR: '[data-testid="add-plan-btn"]',
      TEXT: '新增規劃'
    },
    ADD_TRADE_BTN: {
      SELECTOR: '[data-testid="add-trade-btn"]',
      TEXT: '新增交易'
    },
    ADD_PLAN_CARD: {
      SELECTOR: 'div.add-card.plan',
      TEXT: '新增規劃'
    },
    ADD_TRADE_CARD: {
      SELECTOR: 'div.add-card.trade',
      TEXT: '新增交易紀錄'
    }
  },
  ACCOUNTS: {
    URL: '/accounts',
    HEADER: {
      SELECTOR: '[data-testid="accounts-header"]',
      TEXT: '交易帳號管理'
    },
    ADD_ACCOUNT_BTN: {
      SELECTOR: '[data-testid="add-account-btn"]',
      TEXT: '新增交易帳號'
    },
    IMPORT_CSV_BTN: {
      SELECTOR: '[data-testid="import-csv-btn"]',
      TEXT: '匯入 CSV'
    },
    CLEAR_DATA_BTN: {
      SELECTOR: '[data-testid="clear-data-btn"]',
      TEXT: '清除資料'
    },
    MODAL_TITLE: {
      SELECTOR: 'h2',
      TEXT: '新增交易帳號'
    }
  }
};
