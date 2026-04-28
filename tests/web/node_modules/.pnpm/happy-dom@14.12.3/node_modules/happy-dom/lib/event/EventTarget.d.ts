import IEventListener from './IEventListener.js';
import * as PropertySymbol from '../PropertySymbol.js';
import Event from './Event.js';
import IEventListenerOptions from './IEventListenerOptions.js';
/**
 * Handles events.
 */
export default abstract class EventTarget {
    #private;
    readonly [PropertySymbol.listeners]: {
        [k: string]: (((event: Event) => void) | IEventListener)[];
    };
    readonly [PropertySymbol.listenerOptions]: {
        [k: string]: (IEventListenerOptions | null)[];
    };
    /**
     * Return a default description for the EventTarget class.
     */
    get [Symbol.toStringTag](): string;
    /**
     * Adds an event listener.
     *
     * @param type Event type.
     * @param listener Listener.
     * @param options An object that specifies characteristics about the event listener.(currently only once)
     * @param options.once
     */
    addEventListener(type: string, listener: ((event: Event) => void) | IEventListener, options?: boolean | IEventListenerOptions): void;
    /**
     * Adds an event listener.
     *
     * @param type Event type.
     * @param listener Listener.
     */
    removeEventListener(type: string, listener: ((event: Event) => void) | IEventListener): void;
    /**
     * Dispatches an event.
     *
     * @see https://www.w3.org/TR/DOM-Level-3-Events/#event-flow
     * @see https://www.quirksmode.org/js/events_order.html#link4
     * @param event Event.
     * @returns The return value is false if event is cancelable and at least one of the event handlers which handled this event called Event.preventDefault().
     */
    dispatchEvent(event: Event): boolean;
    /**
     * Adds an event listener.
     *
     * TODO:
     * Was used by with IE8- and Opera. React believed Happy DOM was a legacy browser and used them, but that is no longer the case, so we should remove this method after that this is verified.
     *
     * @deprecated
     * @param type Event type.
     * @param listener Listener.
     */
    attachEvent(type: string, listener: ((event: Event) => void) | IEventListener): void;
    /**
     * Removes an event listener.
     *
     * TODO:
     * Was used by IE8- and Opera. React believed Happy DOM was a legacy browser and used them, but that is no longer the case, so we should remove this method after that this is verified.
     *
     * @deprecated
     * @param type Event type.
     * @param listener Listener.
     */
    detachEvent(type: string, listener: ((event: Event) => void) | IEventListener): void;
}
//# sourceMappingURL=EventTarget.d.ts.map