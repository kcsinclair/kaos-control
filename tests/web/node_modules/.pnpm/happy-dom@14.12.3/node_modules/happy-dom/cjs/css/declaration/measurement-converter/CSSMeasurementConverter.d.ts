import BrowserWindow from '../../../window/BrowserWindow.cjs';
/**
 * CSS Measurement Converter.
 */
export default class CSSMeasurementConverter {
    /**
     * Returns measurement in pixels.
     *
     * @param options Options.
     * @param options.ownerWindow Owner window.
     * @param options.value Measurement (e.g. "10px", "10rem" or "10em").
     * @param options.rootFontSize Root font size in pixels.
     * @param options.parentFontSize Parent font size in pixels.
     * @param [options.parentSize] Parent size in pixels.
     * @returns Measurement in pixels.
     */
    static toPixels(options: {
        ownerWindow: BrowserWindow;
        value: string;
        rootFontSize: string | number;
        parentFontSize: string | number;
        parentSize?: string | number | null;
    }): number | null;
    /**
     * Rounds the number with 4 decimals.
     *
     * @param value Value.
     * @returns Rounded value.
     */
    static round(value: number): number;
}
//# sourceMappingURL=CSSMeasurementConverter.d.ts.map