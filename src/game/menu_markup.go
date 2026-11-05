// menu_markup.go holds the Kaiju HTML/CSS-like markup documents for the
// production main-menu screens (Story 008 boot wiring, ADR-0012). The
// documents are built once at boot and never mutated at runtime: every
// dynamic string is a static label pool bound per frame via InnerLabel(),
// every interactive element carries a fixed id whose Go-side click handler
// dispatches a mouse activation through the screen flow (the screens' focus
// managers own the ids). Visibility is entity Activate/Deactivate — never
// CSS display toggles (this engine version does not re-apply layout on
// display changes) and never runtime tree mutations (anyChildDirty panic).
//
// Layout is absolute PX positioning against the 1280x720 reference window
// (the engine default; the minimum supported resolution). Palette follows
// the art bible desk register: soot #141210, parchment #E3D5B0, ink
// #1C1813, amber #D9A441, oxblood #7A2E28.
package game

import (
	"fmt"
	"strings"
)

// settingsRowSlots is the static settings-row label pool size. The largest
// category is Accessibility (14 rows + re-run + reset-category = 16), so 16
// slots cover every category's RenderCommands stream.
const settingsRowSlots = 16

// settingsRowSlotHeightPx is the vertical pitch of a settings row slot.
const settingsRowSlotHeightPx = 31

// keybindRowSlots is the static keybinding-row pool size: 14 actions in the
// default CR-6 table (4 movement + 4 camera + 2 combat + 4 UI).
const keybindRowSlots = 14

// keybindRowSlotHeightPx is the vertical pitch of a keybinding row.
const keybindRowSlotHeightPx = 26

// zeroWidthSpace is the markup text content of every bound label: the label
// element must exist with content for InnerLabel to return it, and the
// zero-width space renders as nothing until Go binds real text.
const zeroWidthSpace = "&#8203;"

// titleScreenFlowHTML is the production title screen: soot backdrop, title
// block, and the four menu buttons plus the difficulty and name-entry inline
// views of the Story 002 controller. Button ids match the TitleScreen focus
// ids so clicks dispatch through the flow's focus manager.
const titleScreenFlowHTML = `<!DOCTYPE html>
<html>
<head>
<style>
body { margin: 0; padding: 0; background-color: #141210; }
#ts-root { position: absolute; top: 0; left: 0; width: 1280px; height: 720px; }
#ts-title { position: absolute; top: 110px; left: 96px; width: 640px; height: 54px;
    color: #E9E6DC; font-size: 36px; }
#ts-subtitle { position: absolute; top: 168px; left: 96px; width: 640px; height: 24px;
    color: #988E84; font-size: 16px; }
.menu-btn { position: absolute; left: 96px; width: 340px; height: 44px;
    background-color: #1C1813; color: #E9E6DC; font-size: 16px; border: 1px solid #2A231C; }
#new-game { top: 240px; }
#load-game { top: 294px; }
#settings { top: 348px; }
#quit { top: 402px; }
#ts-version { position: absolute; right: 16px; bottom: 12px; color: rgba(233, 230, 220, 0.6);
    font-size: 13px; }
#ts-diff-title { position: absolute; top: 200px; left: 96px; width: 640px; height: 30px;
    color: #E9E6DC; font-size: 22px; }
#difficulty-narrative { top: 250px; }
#difficulty-normal { top: 304px; }
#difficulty-tactical { top: 358px; }
#ts-name-title { position: absolute; top: 200px; left: 96px; width: 640px; height: 30px;
    color: #E9E6DC; font-size: 22px; }
#ts-name-label { position: absolute; top: 250px; left: 96px; width: 640px; height: 30px;
    color: #D9A441; font-size: 20px; }
#name-begin { top: 320px; }
#name-cancel { top: 374px; }
</style>
</head>
<body>
<div id="ts-root">
    <Label id="ts-title">The Marrow of Empire</Label>
    <Label id="ts-subtitle">An Isometric Investigation</Label>

    <div id="ts-menu-view">
        <Button id="new-game" class="menu-btn">&#8203;</Button>
        <Button id="load-game" class="menu-btn">&#8203;</Button>
        <Button id="settings" class="menu-btn">&#8203;</Button>
        <Button id="quit" class="menu-btn">&#8203;</Button>
    </div>

    <div id="ts-diff-view">
        <Label id="ts-diff-title">Choose a Difficulty</Label>
        <Button id="difficulty-narrative" class="menu-btn">&#8203;</Button>
        <Button id="difficulty-normal" class="menu-btn">&#8203;</Button>
        <Button id="difficulty-tactical" class="menu-btn">&#8203;</Button>
    </div>

    <div id="ts-name-view">
        <Label id="ts-name-title">Name Your Wayfarer</Label>
        <Label id="ts-name-label">&#8203;</Label>
        <Button id="name-begin" class="menu-btn">&#8203;</Button>
        <Button id="name-cancel" class="menu-btn">&#8203;</Button>
    </div>

    <Label id="ts-version">&#8203;</Label>
</div>
</body>
</html>`

// settingsScreenFlowHTML is the settings screen: six category tabs, a static
// pool of settingsRowSlots rows (label button + value label), the footer
// (reset category / reset all / back), and the reset dialog, preview overlay,
// and toast. Row ids are positional ("settings-row-N"); the Go pass stores
// each slot's current focus id (per-frame render command FocusID) and the
// slot click handler dispatches that id.
const settingsScreenFlowHTML = `<!DOCTYPE html>
<html>
<head>
<style>
body { margin: 0; padding: 0; background-color: #141210; }
#st-root { position: absolute; top: 0; left: 0; width: 1280px; height: 720px; }
#st-panel { position: absolute; top: 40px; left: 40px; width: 660px; height: 660px;
    background-color: #E3D5B0; border: 1px solid rgba(28, 24, 19, 0.25); z-index: 1; }
#st-title { position: absolute; top: 12px; left: 20px; width: 300px; height: 26px;
    color: #1C1813; font-size: 20px; }
.tab-btn { position: absolute; top: 46px; width: 100px; height: 32px;
    background-color: transparent; color: #1C1813; font-size: 14px; }
#tab-gameplay { left: 12px; }
#tab-camera { left: 118px; }
#tab-audio { left: 224px; }
#tab-controls { left: 330px; }
#tab-accessibility { left: 436px; }
#tab-graphics { left: 542px; }
.tab-btn.selected { border-bottom: 2px solid #D9A441; font-weight: bold; }
#st-rows { position: absolute; top: 90px; left: 12px; width: 636px; height: 500px; }
.setting-row { position: absolute; left: 0; width: 636px; height: 26px;
    background-color: #EDE3C8; color: #1C1813; font-size: 14px; border: 1px solid rgba(28, 24, 19, 0.15); }
.setting-row-value { position: absolute; right: 8px; top: 4px; width: 220px; height: 18px;
    color: #5A4F42; font-size: 13px; }
#settings-row-0 { top: 0px; }
#settings-row-1 { top: 31px; }
#settings-row-2 { top: 62px; }
#settings-row-3 { top: 93px; }
#settings-row-4 { top: 124px; }
#settings-row-5 { top: 155px; }
#settings-row-6 { top: 186px; }
#settings-row-7 { top: 217px; }
#settings-row-8 { top: 248px; }
#settings-row-9 { top: 279px; }
#settings-row-10 { top: 310px; }
#settings-row-11 { top: 341px; }
#settings-row-12 { top: 372px; }
#settings-row-13 { top: 403px; }
#settings-row-14 { top: 434px; }
#settings-row-15 { top: 465px; }
#st-footer { position: absolute; top: 600px; left: 12px; width: 636px; height: 40px; }
.footer-btn { position: absolute; top: 0; width: 200px; height: 32px;
    background-color: #1C1813; color: #E9E6DC; font-size: 14px; }
#settings-reset-category { left: 0; }
#settings-reset-all { left: 210px; }
#settings-back { left: 420px; }
#st-dialog { position: absolute; top: 0; left: 0; width: 1280px; height: 720px;
    background-color: rgba(20, 18, 16, 0.6); z-index: 2; }
#st-dialog-panel { position: absolute; top: 260px; left: 300px; width: 520px; height: 180px;
    background-color: #E3D5B0; border: 1px solid rgba(28, 24, 19, 0.25); }
#st-dialog-msg { position: absolute; top: 20px; left: 20px; width: 480px; height: 60px;
    color: #1C1813; font-size: 15px; }
.dialog-btn { position: absolute; top: 120px; width: 160px; height: 36px;
    background-color: #1C1813; color: #E9E6DC; font-size: 14px; }
#settings-reset-confirm { left: 60px; }
#settings-reset-cancel { left: 260px; }
#st-preview { position: absolute; top: 0; left: 0; width: 1280px; height: 720px;
    background-color: rgba(20, 18, 16, 0.6); z-index: 2; }
#st-preview-panel { position: absolute; top: 300px; left: 360px; width: 440px; height: 120px;
    background-color: #E3D5B0; border: 1px solid rgba(28, 24, 19, 0.25); }
#st-preview-msg { position: absolute; top: 16px; left: 20px; width: 400px; height: 30px;
    color: #1C1813; font-size: 15px; }
#settings-preview-keep { left: 40px; top: 64px; }
#settings-preview-revert { left: 220px; top: 64px; }
#st-toast { position: absolute; bottom: 30px; left: 380px; width: 460px; height: 44px;
    background-color: #7A2E28; color: #E9E6DC; font-size: 14px; z-index: 3; }
#toast-dismiss { position: absolute; right: 8px; top: 6px; width: 90px; height: 30px;
    background-color: #1C1813; color: #E9E6DC; font-size: 13px; }
</style>
</head>
<body>
<div id="st-root">
    <div id="st-panel">
        <Label id="st-title">Settings</Label>
        <Button id="tab-gameplay" class="tab-btn">&#8203;</Button>
        <Button id="tab-camera" class="tab-btn">&#8203;</Button>
        <Button id="tab-audio" class="tab-btn">&#8203;</Button>
        <Button id="tab-controls" class="tab-btn">&#8203;</Button>
        <Button id="tab-accessibility" class="tab-btn">&#8203;</Button>
        <Button id="tab-graphics" class="tab-btn">&#8203;</Button>
        <div id="st-rows">
            <Button id="settings-row-0" class="setting-row">&#8203;<Label id="settings-row-0-value" class="setting-row-value">&#8203;</Label></Button>
            <Button id="settings-row-1" class="setting-row">&#8203;<Label id="settings-row-1-value" class="setting-row-value">&#8203;</Label></Button>
            <Button id="settings-row-2" class="setting-row">&#8203;<Label id="settings-row-2-value" class="setting-row-value">&#8203;</Label></Button>
            <Button id="settings-row-3" class="setting-row">&#8203;<Label id="settings-row-3-value" class="setting-row-value">&#8203;</Label></Button>
            <Button id="settings-row-4" class="setting-row">&#8203;<Label id="settings-row-4-value" class="setting-row-value">&#8203;</Label></Button>
            <Button id="settings-row-5" class="setting-row">&#8203;<Label id="settings-row-5-value" class="setting-row-value">&#8203;</Label></Button>
            <Button id="settings-row-6" class="setting-row">&#8203;<Label id="settings-row-6-value" class="setting-row-value">&#8203;</Label></Button>
            <Button id="settings-row-7" class="setting-row">&#8203;<Label id="settings-row-7-value" class="setting-row-value">&#8203;</Label></Button>
            <Button id="settings-row-8" class="setting-row">&#8203;<Label id="settings-row-8-value" class="setting-row-value">&#8203;</Label></Button>
            <Button id="settings-row-9" class="setting-row">&#8203;<Label id="settings-row-9-value" class="setting-row-value">&#8203;</Label></Button>
            <Button id="settings-row-10" class="setting-row">&#8203;<Label id="settings-row-10-value" class="setting-row-value">&#8203;</Label></Button>
            <Button id="settings-row-11" class="setting-row">&#8203;<Label id="settings-row-11-value" class="setting-row-value">&#8203;</Label></Button>
            <Button id="settings-row-12" class="setting-row">&#8203;<Label id="settings-row-12-value" class="setting-row-value">&#8203;</Label></Button>
            <Button id="settings-row-13" class="setting-row">&#8203;<Label id="settings-row-13-value" class="setting-row-value">&#8203;</Label></Button>
            <Button id="settings-row-14" class="setting-row">&#8203;<Label id="settings-row-14-value" class="setting-row-value">&#8203;</Label></Button>
            <Button id="settings-row-15" class="setting-row">&#8203;<Label id="settings-row-15-value" class="setting-row-value">&#8203;</Label></Button>
        </div>
        <div id="st-footer">
            <Button id="settings-reset-category" class="footer-btn">&#8203;</Button>
            <Button id="settings-reset-all" class="footer-btn">&#8203;</Button>
            <Button id="settings-back" class="footer-btn">&#8203;</Button>
        </div>
    </div>

    <div id="st-dialog">
        <div id="st-dialog-panel">
            <Label id="st-dialog-msg">&#8203;</Label>
            <Button id="settings-reset-confirm" class="dialog-btn">&#8203;</Button>
            <Button id="settings-reset-cancel" class="dialog-btn">&#8203;</Button>
        </div>
    </div>

    <div id="st-preview">
        <div id="st-preview-panel">
            <Label id="st-preview-msg">&#8203;</Label>
            <Button id="settings-preview-keep" class="dialog-btn">&#8203;</Button>
            <Button id="settings-preview-revert" class="dialog-btn">&#8203;</Button>
        </div>
    </div>

    <div id="st-toast">
        <Label id="st-toast-msg">&#8203;</Label>
        <Button id="toast-dismiss">&#8203;</Button>
    </div>
</div>
</body>
</html>`

// keybindingsScreenFlowHTML is the keybindings screen: one static row per
// action (label, primary slot, secondary slot, restore), the gamepad preset
// row, the footer (restore all / back), the reserved-key warning dialog, the
// restore-all confirmation, and the AC-6 blocked banner. Row ids are
// positional ("kb-row-N-*"); the Go pass keeps the action's focus ids
// (keybindFocusID) per row and dispatches those on click.
const keybindingsScreenFlowHTML = `<!DOCTYPE html>
<html>
<head>
<style>
body { margin: 0; padding: 0; background-color: #141210; }
#kb-root { position: absolute; top: 0; left: 0; width: 1280px; height: 720px; }
#kb-panel { position: absolute; top: 40px; left: 40px; width: 700px; height: 640px;
    background-color: #E3D5B0; border: 1px solid rgba(28, 24, 19, 0.25); z-index: 1; }
#kb-title { position: absolute; top: 12px; left: 20px; width: 300px; height: 26px;
    color: #1C1813; font-size: 20px; }
#kb-preset-row { position: absolute; top: 44px; left: 12px; width: 676px; height: 28px; }
#kb-preset-label { position: absolute; left: 0; top: 4px; width: 200px; height: 20px;
    color: #1C1813; font-size: 14px; }
#kb-gamepad-preset { position: absolute; right: 0; top: 0; width: 240px; height: 26px;
    background-color: #EDE3C8; color: #1C1813; font-size: 14px; border: 1px solid rgba(28, 24, 19, 0.15); }
#kb-blocked { position: absolute; top: 74px; left: 12px; width: 676px; height: 20px;
    color: #7A2E28; font-size: 13px; }
#kb-rows { position: absolute; top: 96px; left: 0; width: 700px; height: 470px; }
.kb-row { position: absolute; left: 12px; width: 676px; height: 24px; }
.kb-row-label { position: absolute; left: 0; top: 3px; width: 190px; height: 18px;
    color: #1C1813; font-size: 13px; }
.kb-slot-btn { position: absolute; top: 0; height: 24px;
    background-color: #EDE3C8; color: #1C1813; font-size: 13px; border: 1px solid rgba(28, 24, 19, 0.15); }
.kb-primary { left: 200px; width: 110px; }
.kb-secondary { left: 318px; width: 110px; }
.kb-restore { left: 436px; width: 70px; }
.kb-restore-btn { position: absolute; top: 0; left: 436px; width: 70px; height: 24px;
    background-color: transparent; color: #5A4F42; font-size: 12px; border: none; }
#kb-row-0 { top: 0px; }
#kb-row-1 { top: 26px; }
#kb-row-2 { top: 52px; }
#kb-row-3 { top: 78px; }
#kb-row-4 { top: 104px; }
#kb-row-5 { top: 130px; }
#kb-row-6 { top: 156px; }
#kb-row-7 { top: 182px; }
#kb-row-8 { top: 208px; }
#kb-row-9 { top: 234px; }
#kb-row-10 { top: 260px; }
#kb-row-11 { top: 286px; }
#kb-row-12 { top: 312px; }
#kb-row-13 { top: 338px; }
#kb-footer { position: absolute; top: 580px; left: 12px; width: 676px; height: 40px; }
.footer-btn { position: absolute; top: 0; width: 220px; height: 32px;
    background-color: #1C1813; color: #E9E6DC; font-size: 14px; }
#kb-restore-all { left: 0; }
#kb-back { left: 230px; }
#kb-dialog { position: absolute; top: 0; left: 0; width: 1280px; height: 720px;
    background-color: rgba(20, 18, 16, 0.6); z-index: 2; }
#kb-dialog-panel { position: absolute; top: 260px; left: 300px; width: 560px; height: 180px;
    background-color: #E3D5B0; border: 1px solid rgba(28, 24, 19, 0.25); }
#kb-dialog-msg { position: absolute; top: 20px; left: 20px; width: 520px; height: 60px;
    color: #1C1813; font-size: 15px; }
.dialog-btn { position: absolute; top: 120px; width: 180px; height: 36px;
    background-color: #1C1813; color: #E9E6DC; font-size: 14px; }
#kb-reserved-bind { left: 60px; }
#kb-reserved-cancel { left: 280px; }
#kb-restoreall-confirm { left: 60px; }
#kb-restoreall-cancel { left: 280px; }
</style>
</head>
<body>
<div id="kb-root">
    <div id="kb-panel">
        <Label id="kb-title">Keybindings</Label>
        <div id="kb-preset-row">
            <Label id="kb-preset-label">&#8203;</Label>
            <Button id="kb-gamepad-preset">&#8203;</Button>
        </div>
        <Label id="kb-blocked">&#8203;</Label>
        <div id="kb-rows">
            <div id="kb-row-0" class="kb-row"><Label id="kb-row-0-label" class="kb-row-label">&#8203;</Label>
                <Button id="kb-row-0-primary" class="kb-slot-btn kb-primary">&#8203;</Button>
                <Button id="kb-row-0-secondary" class="kb-slot-btn kb-secondary">&#8203;</Button>
                <Button id="kb-row-0-restore" class="kb-restore-btn">&#8203;</Button></div>
            <div id="kb-row-1" class="kb-row"><Label id="kb-row-1-label" class="kb-row-label">&#8203;</Label>
                <Button id="kb-row-1-primary" class="kb-slot-btn kb-primary">&#8203;</Button>
                <Button id="kb-row-1-secondary" class="kb-slot-btn kb-secondary">&#8203;</Button>
                <Button id="kb-row-1-restore" class="kb-restore-btn">&#8203;</Button></div>
            <div id="kb-row-2" class="kb-row"><Label id="kb-row-2-label" class="kb-row-label">&#8203;</Label>
                <Button id="kb-row-2-primary" class="kb-slot-btn kb-primary">&#8203;</Button>
                <Button id="kb-row-2-secondary" class="kb-slot-btn kb-secondary">&#8203;</Button>
                <Button id="kb-row-2-restore" class="kb-restore-btn">&#8203;</Button></div>
            <div id="kb-row-3" class="kb-row"><Label id="kb-row-3-label" class="kb-row-label">&#8203;</Label>
                <Button id="kb-row-3-primary" class="kb-slot-btn kb-primary">&#8203;</Button>
                <Button id="kb-row-3-secondary" class="kb-slot-btn kb-secondary">&#8203;</Button>
                <Button id="kb-row-3-restore" class="kb-restore-btn">&#8203;</Button></div>
            <div id="kb-row-4" class="kb-row"><Label id="kb-row-4-label" class="kb-row-label">&#8203;</Label>
                <Button id="kb-row-4-primary" class="kb-slot-btn kb-primary">&#8203;</Button>
                <Button id="kb-row-4-secondary" class="kb-slot-btn kb-secondary">&#8203;</Button>
                <Button id="kb-row-4-restore" class="kb-restore-btn">&#8203;</Button></div>
            <div id="kb-row-5" class="kb-row"><Label id="kb-row-5-label" class="kb-row-label">&#8203;</Label>
                <Button id="kb-row-5-primary" class="kb-slot-btn kb-primary">&#8203;</Button>
                <Button id="kb-row-5-secondary" class="kb-slot-btn kb-secondary">&#8203;</Button>
                <Button id="kb-row-5-restore" class="kb-restore-btn">&#8203;</Button></div>
            <div id="kb-row-6" class="kb-row"><Label id="kb-row-6-label" class="kb-row-label">&#8203;</Label>
                <Button id="kb-row-6-primary" class="kb-slot-btn kb-primary">&#8203;</Button>
                <Button id="kb-row-6-secondary" class="kb-slot-btn kb-secondary">&#8203;</Button>
                <Button id="kb-row-6-restore" class="kb-restore-btn">&#8203;</Button></div>
            <div id="kb-row-7" class="kb-row"><Label id="kb-row-7-label" class="kb-row-label">&#8203;</Label>
                <Button id="kb-row-7-primary" class="kb-slot-btn kb-primary">&#8203;</Button>
                <Button id="kb-row-7-secondary" class="kb-slot-btn kb-secondary">&#8203;</Button>
                <Button id="kb-row-7-restore" class="kb-restore-btn">&#8203;</Button></div>
            <div id="kb-row-8" class="kb-row"><Label id="kb-row-8-label" class="kb-row-label">&#8203;</Label>
                <Button id="kb-row-8-primary" class="kb-slot-btn kb-primary">&#8203;</Button>
                <Button id="kb-row-8-secondary" class="kb-slot-btn kb-secondary">&#8203;</Button>
                <Button id="kb-row-8-restore" class="kb-restore-btn">&#8203;</Button></div>
            <div id="kb-row-9" class="kb-row"><Label id="kb-row-9-label" class="kb-row-label">&#8203;</Label>
                <Button id="kb-row-9-primary" class="kb-slot-btn kb-primary">&#8203;</Button>
                <Button id="kb-row-9-secondary" class="kb-slot-btn kb-secondary">&#8203;</Button>
                <Button id="kb-row-9-restore" class="kb-restore-btn">&#8203;</Button></div>
            <div id="kb-row-10" class="kb-row"><Label id="kb-row-10-label" class="kb-row-label">&#8203;</Label>
                <Button id="kb-row-10-primary" class="kb-slot-btn kb-primary">&#8203;</Button>
                <Button id="kb-row-10-secondary" class="kb-slot-btn kb-secondary">&#8203;</Button>
                <Button id="kb-row-10-restore" class="kb-restore-btn">&#8203;</Button></div>
            <div id="kb-row-11" class="kb-row"><Label id="kb-row-11-label" class="kb-row-label">&#8203;</Label>
                <Button id="kb-row-11-primary" class="kb-slot-btn kb-primary">&#8203;</Button>
                <Button id="kb-row-11-secondary" class="kb-slot-btn kb-secondary">&#8203;</Button>
                <Button id="kb-row-11-restore" class="kb-restore-btn">&#8203;</Button></div>
            <div id="kb-row-12" class="kb-row"><Label id="kb-row-12-label" class="kb-row-label">&#8203;</Label>
                <Button id="kb-row-12-primary" class="kb-slot-btn kb-primary">&#8203;</Button>
                <Button id="kb-row-12-secondary" class="kb-slot-btn kb-secondary">&#8203;</Button>
                <Button id="kb-row-12-restore" class="kb-restore-btn">&#8203;</Button></div>
            <div id="kb-row-13" class="kb-row"><Label id="kb-row-13-label" class="kb-row-label">&#8203;</Label>
                <Button id="kb-row-13-primary" class="kb-slot-btn kb-primary">&#8203;</Button>
                <Button id="kb-row-13-secondary" class="kb-slot-btn kb-secondary">&#8203;</Button>
                <Button id="kb-row-13-restore" class="kb-restore-btn">&#8203;</Button></div>
        </div>
        <div id="kb-footer">
            <Button id="kb-restore-all" class="footer-btn">&#8203;</Button>
            <Button id="kb-back" class="footer-btn">&#8203;</Button>
        </div>
    </div>

    <div id="kb-dialog">
        <div id="kb-dialog-panel">
            <Label id="kb-dialog-msg">&#8203;</Label>
            <Button id="kb-reserved-bind" class="dialog-btn">&#8203;</Button>
            <Button id="kb-reserved-cancel" class="dialog-btn">&#8203;</Button>
            <Button id="kb-restoreall-confirm" class="dialog-btn">&#8203;</Button>
            <Button id="kb-restoreall-cancel" class="dialog-btn">&#8203;</Button>
        </div>
    </div>
</div>
</body>
</html>`

// keybindFocusID builds a slot focus id for an action row
// ("move_forward" + "primary" -> "kb-row-move-forward-primary"), matching the
// keybindings screen's FocusPrimaryIDFmt/FocusSecondaryIDFmt/FocusRestoreIDFmt
// contract (kebab-case action ID).
func keybindFocusID(actionID, slot string) string {
	return fmt.Sprintf("kb-row-%s-%s", strings.ReplaceAll(actionID, "_", "-"), slot)
}

// pauseMenuFlowHTML is the pause menu overlay: the world dim (Desk register)
// and the five options (Resume, Save, Load, Settings, Quit), plus the quit
// confirmation dialog. The world-freeze contract (Story 006) is out of scope
// for this wiring pass; the overlay renders and routes input only.
const pauseMenuFlowHTML = `<!DOCTYPE html>
<html>
<head>
<style>
body { margin: 0; padding: 0; }
#pm-dim { position: absolute; top: 0; left: 0; width: 1280px; height: 720px;
    background-color: rgba(20, 18, 16, 0.72); z-index: 10; }
#pm-panel { position: absolute; top: 180px; left: 440px; width: 400px; height: 360px;
    background-color: #E3D5B0; border: 1px solid rgba(28, 24, 19, 0.25); z-index: 11; }
#pm-title { position: absolute; top: 16px; left: 20px; width: 360px; height: 26px;
    color: #1C1813; font-size: 20px; }
.pm-btn { position: absolute; left: 60px; width: 280px; height: 40px;
    background-color: #1C1813; color: #E9E6DC; font-size: 15px; }
#resume { top: 64px; }
#save { top: 112px; }
#load { top: 160px; }
#settings { top: 208px; }
#quit { top: 256px; }
#pm-confirm { position: absolute; top: 0; left: 0; width: 1280px; height: 720px;
    background-color: rgba(20, 18, 16, 0.6); z-index: 12; }
#pm-confirm-panel { position: absolute; top: 240px; left: 380px; width: 520px; height: 200px;
    background-color: #E3D5B0; border: 1px solid rgba(28, 24, 19, 0.25); }
#pm-confirm-msg { position: absolute; top: 24px; left: 20px; width: 480px; height: 50px;
    color: #1C1813; font-size: 15px; }
#save-and-quit { top: 110px; }
#quit-without-saving { top: 110px; }
#cancel { left: 250px; top: 110px; }
</style>
</head>
<body>
<div id="pm-dim"></div>
<div id="pm-panel">
    <Label id="pm-title">Paused</Label>
    <Button id="resume" class="pm-btn">&#8203;</Button>
    <Button id="save" class="pm-btn">&#8203;</Button>
    <Button id="load" class="pm-btn">&#8203;</Button>
    <Button id="settings" class="pm-btn">&#8203;</Button>
    <Button id="quit" class="pm-btn">&#8203;</Button>
</div>
<div id="pm-confirm">
    <div id="pm-confirm-panel">
        <Label id="pm-confirm-msg">&#8203;</Label>
        <Button id="save-and-quit" class="pm-btn">&#8203;</Button>
        <Button id="quit-without-saving" class="pm-btn">&#8203;</Button>
        <Button id="cancel" class="pm-btn">&#8203;</Button>
    </div>
</div>
</body>
</html>`
