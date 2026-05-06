/**
 * Central widget registration.
 * Import this file once (in main.ts) to register all dashboard widgets.
 * To add a new widget: call registerWidget() here — no other files need editing.
 */
import { registerWidget } from './widgetRegistry'
import SummaryCountsWidget from './widgets/SummaryCountsWidget.vue'

registerWidget('summary-counts', SummaryCountsWidget, { slot: 'summary', order: 0 })
