import MediaQueryItem from './MediaQueryItem.js';
import BrowserWindow from '../window/BrowserWindow.js';
/**
 * Utility for parsing a query string.
 */
export default class MediaQueryParser {
    /**
     * Parses a media query string.
     *
     * @param options Options.
     * @param options.ownerWindow Owner window.
     * @param options.mediaQuery Media query string.
     * @param [options.rootFontSize] Root font size.
     * @returns Media query items.
     */
    static parse(options: {
        ownerWindow: BrowserWindow;
        mediaQuery: string;
        rootFontSize?: string | number | null;
    }): MediaQueryItem[];
}
//# sourceMappingURL=MediaQueryParser.d.ts.map