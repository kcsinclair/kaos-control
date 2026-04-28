import IBrowserSettings from '../browser/types/IBrowserSettings.cjs';
import BrowserWindow from './BrowserWindow.cjs';
/**
 * Browser settings reader that will allow to read settings more securely as it is not possible to override a settings object to make DOM functionality act on it.
 */
export default class WindowBrowserSettingsReader {
    #private;
    /**
     * Returns browser settings.
     *
     * @param window Window.
     * @returns Settings.
     */
    static getSettings(window: BrowserWindow): IBrowserSettings | null;
    /**
     * Sets browser settings.
     *
     * @param window Window.
     * @param settings Settings.
     */
    static setSettings(window: BrowserWindow, settings: IBrowserSettings): void;
    /**
     * Removes browser settings.
     *
     * @param window Window.
     */
    static removeSettings(window: BrowserWindow): void;
}
//# sourceMappingURL=WindowBrowserSettingsReader.d.ts.map