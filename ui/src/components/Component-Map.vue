<template>
  <div id="main">
    <!-- Popup -->
    <div class="popup">
      <div
        ref="popup"
        class="popuptext"
        v-bind:class="{
          show:
            isPopupVisible &&
            isMapLoaded &&
            isInProgress == false &&
            isBlured !== 'true',
        }"
      >
        <popupControl
          :location="selectedPopupLocation"
          :onConnect="connect"
          :onDisconnect="disconnect"
          :onMouseClick="onPopupMouseClick"
          :onResume="resume"
        />
      </div>
    </div>

    <!-- Buttons panel LEFT-->
    <div class="buttonsPanelTopLeft" v-if="isBlured !== 'true'">
      <button class="settingsBtn" v-on:click="onMinimize">
        <img src="@/assets/minimize.svg" />
      </button>
    </div>

    <!-- Buttons panel RIGHT-->
    <div
      class="buttonsPanelTopRight"
      v-bind:class="{
        buttonsPanelTopRightNoFrameWindow: !isWindowHasFrame,
      }"
      v-if="isBlured !== 'true'"
    >
      <button class="settingsBtn settingsBtnMarginLeft" v-on:click="onSettings">
        <img src="@/assets/settings.svg" />
      </button>

      <button
        class="settingsBtn settingsBtnMarginLeft"
        v-on:click="onAccountSettings"
      >
        <img src="@/assets/user.svg" />
      </button>

      <button class="settingsBtn" v-on:click="centerCurrentLocation()">
        <img src="@/assets/crosshair.svg" />
      </button>
    </div>

    <!-- Bottom panel -->
    <!--
    <div class="bottomButtonsPanel" v-if="isBlured !== 'true'">
      <button class="settingsBtn" v-on:click="centerCurrentLocation(false)">
        <img src="@/assets/crosshair.svg" />
      </button>
    </div>
    -->

    <div class="bottomPanel">
      <div class="flexRow">
        <div class="geolocationInfoPanel">
          <GeolocationInfoControl style="display: flex; align-items: center" />
        </div>
        <div calsss="flexRowRestSpace"></div>
      </div>
      <div
        class="accountWillExpire"
        v-if="$store.getters['account/messageAccountExpiration']"
      >
        <img src="@/assets/alert-triangle.svg" style="margin-right: 12px" />
        {{ $store.getters["account/messageAccountExpiration"] }}
        <div class="flexRowRestSpace" />
        <button class="noBordersBtn" v-on:click="onAccountRenew()">
          RENEW
        </button>
      </div>
      <div
        class="trialWillExpire"
        v-on:click="onAccountRenew()"
        v-if="$store.getters['account/messageFreeTrial']"
      >
        <img src="@/assets/alert-circle.svg" style="margin-right: 12px" />
        {{ $store.getters["account/messageFreeTrial"] }}
        <div class="flexRowRestSpace" />
        <button class="noBordersBtn">UPGRADE</button>
      </div>
    </div>
    <!-- Map -->
    <div class="mapcontainer" ref="combined">
      <canvas
        class="canvas"
        ref="canvas"
        v-bind:class="{ blured: isBlured === 'true' }"
        @mousedown="mouseDown"
        @mouseup="mouseUp"
        @mousemove="mouseMove"
        @wheel="wheel"
      >
      </canvas>

      <!-- Top-located canvas to be able to blure map -->
      <canvas
        class="canvasTop"
        v-bind:class="{ blured: isBlured === 'true' }"
      ></canvas>

      <div class="bigMapArea" ref="bigMapArea">
        <img ref="map" class="map" :src="mapImage" @load="mapLoaded" />

        <!-- Hidden element to calculate styled text size-->
        <div
          ref="hiddenTestTextMeter"
          class="mapLocationName"
          style="opacity: 0; pointer-events: none; z-index: -1"
        ></div>

        <div class="mapLocationsContainer" ref="mapLocationsContainer">
          <!-- Location point -->
          <div
            v-show="isMapLoaded"
            class="mapLocationPoint"
            v-for="l of locationsToDisplay"
            @wheel="wheel"
            v-on:click="locationClicked(l.location)"
            v-bind:key="'point_' + l.location.city"
            v-bind:class="{
              mapLocationPointCurrent: l.location === location,
              mapLocationPointConnected:
                l.location === connectedLocation && isConnected,
            }"
            :style="{
              left: l.x - l.pointRadius + 'px',
              top: l.y - l.pointRadius + 'px',
              height: l.pointRadius * 2 + 'px',
              width: l.pointRadius * 2 + 'px',
            }"
          ></div>
          <!-- Location name -->
          <div
            v-show="isMapLoaded"
            class="mapLocationName"
            v-for="l of locationsToDisplay"
            @wheel="wheel"
            v-on:click="locationClicked(l.location)"
            v-bind:key="'name_' + l.location.city"
            v-bind:class="{
              mapLocationNameCurrent: l.location === location,
              mapLocationNameConnected:
                l.location === connectedLocation && isConnected,
            }"
            :style="{ left: l.left + 'px', top: l.top + 'px' }"
          >
            {{ l.width > 0 ? l.location.city : "" }}
          </div>

          <!-- Animation elements -->
          <div
            v-show="isMapLoaded"
            ref="animationCurrLoactionCirecle"
            class="mapLocationCircleCurrLocation"
          ></div>

          <div
            v-show="isMapLoaded"
            ref="animationSelectedCirecle1"
            class="mapLocationCircleSelectedSvr"
          ></div>
          <div
            v-show="isMapLoaded"
            ref="animationSelectedCirecle2"
            class="mapLocationCircleSelectedSvr"
          ></div>

          <div
            v-show="isMapLoaded"
            ref="animationConnectedWaves"
            class="mapLocationCircleConnectedWaves"
            v-bind:class="{
              mapLocationCircleConnectedWavesRunning: isConnected,
            }"
          ></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { VpnStateEnum, PauseStateEnum, ColorTheme } from "@/store/types";
import {
  IsOsDarkColorScheme,
  CheckAndNotifyInaccessibleServer,
} from "@/helpers/renderer";

const sender = window.ipcSender;
import popupControl from "@/components/controls/control-map-popup.vue";
import GeolocationInfoControl from "@/components/controls/control-geolocation-info.vue";
import { IsWindowHasFrame } from "@/platform/platform";

import Image_world_map_dark from "@/assets/world_map_dark.svg";
import Image_world_map_light from "@/assets/world_map_light.svg";

import config from "@/config";

import {
  notLinear,
  getPosFromCoordinates,
  getCoordinatesBy,
} from "@/helpers/helpers";

const defaultZoomScale = 0.41;
export default {
  components: {
    popupControl,
    GeolocationInfoControl,
  },
  props: {
    isBlured: String,
    onAccountSettings: Function,
    onSettings: Function,
    onMinimize: Function,
  },

  data: () => ({
    isDarkTheme: false,

    selectedPopupLocation: null,
    isMapLoaded: false,
    isPopupVisible: false,

    canvas: null,

    bigMapArea: null,
    map: null,
    combinedDiv: null,
    popup: null,
    mapLocationsContainer: null,
    hiddenTestTextMeter: null,

    // refs to two animation circles. First can have 'grow' style.
    // The second must have 'shrink' style.
    animSelCircles: [],
    animCurrLocCircle: null,
    animConnectedWaves: null,

    scale: defaultZoomScale,

    scrollTo: {
      left: -1,
      top: -1,
      isPopupRequired: false,
      startTimeMs: null,
      startLeft: -1,
      startTop: -1,
      maxDurationMs: 0,
    },

    moving: false,
    startMoveX: 0,
    startMoveY: 0,
    lastMoveX: 0,
    lastMoveY: 0,

    mapPos: {
      mapLeftOffset: 0.0,
      mapTopOffset: 0.0,
    },

    locationsToDisplay: [
      /*{
        location,
        x,
        y,
        left: textPos.textX,
        top: textPos.textY,
        pointRadius,
        circleRadius,
        width: textWidth,
        height: textHeight
      } */
    ],
  }),

  computed: {
    isWindowHasFrame: function () {
      return IsWindowHasFrame();
    },
    mapImage: function () {
      if (this.isDarkTheme) return Image_world_map_dark;
      return Image_world_map_light;
    },
    servers: function () {
      return this.$store.getters["vpnState/activeServers"];
    },
    entryServer: function () {
      return this.$store.state.settings.serverEntry;
    },
    selectedServer: function () {
      if (this.$store.state.settings.isMultiHop)
        return this.$store.state.settings.serverExit;
      return this.$store.state.settings.serverEntry;
    },

    location: function () {
      let l = this.$store.state.location;

      // IPv6
      if (this.$store.getters["getIsIPv6View"]) {
        let lv6 = this.$store.state.locationIPv6;
        if (lv6 && lv6.isRealLocation) l = lv6;
        if (!lv6) l = null;
      }

      if (l == null || l.isRealLocation !== true) return null;
      return l;
    },

    isRequestingLocation: function () {
      let isInProcess = this.$store.state.isRequestingLocation;
      // IPv6
      if (this.$store.getters["getIsIPv6View"]) {
        isInProcess = this.$store.state.isRequestingLocationIPv6;
      }
      return isInProcess;
    },

    isFastestServer: function () {
      return this.$store.state.settings.isFastestServer;
    },
    isRandomExitServer: function () {
      return this.$store.state.settings.isRandomExitServer;
    },

    isPaused: function () {
      return this.$store.state.vpnState.pauseState === PauseStateEnum.Paused;
    },

    connectedLocation: function () {
      if (!this.isConnected) return null;
      return this.selectedServer;
    },

    // needed for watcher
    isLoggedIn: function () {
      return this.$store.getters["account/isLoggedIn"];
    },
    // needed for watcher
    connectionState: function () {
      return this.$store.state.vpnState.connectionState;
    },
    isConnecting: function () {
      return this.$store.getters["vpnState/isConnecting"];
    },
    isConnected: function () {
      if (this.isPaused) return false;
      return (
        this.$store.state.vpnState.connectionState === VpnStateEnum.CONNECTED
      );
    },
    isDisconnected: function () {
      if (this.isPaused) return true;
      return (
        this.$store.state.vpnState.connectionState === VpnStateEnum.DISCONNECTED
      );
    },
    isInProgress: function () {
      return !this.isConnected && !this.isDisconnected;
    },
    isMinimizedUI: function () {
      return this.$store.state.settings.minimizedUI;
    },
  },

  mounted() {
    // COLOR SCHEME
    window.matchMedia("(prefers-color-scheme: dark)").addListener(() => {
      this.updateColorScheme();
    });
    this.updateColorScheme();

    this.canvas = this.$refs.canvas;

    this.popup = this.$refs.popup;
    this.bigMapArea = this.$refs.bigMapArea;
    this.map = this.$refs.map;
    this.combinedDiv = this.$refs.combined;
    this.mapLocationsContainer = this.$refs.mapLocationsContainer;
    this.hiddenTestTextMeter = this.$refs.hiddenTestTextMeter;

    this.animSelCircles.push(this.$refs.animationSelectedCirecle1);
    this.animSelCircles.push(this.$refs.animationSelectedCirecle2);
    this.animCurrLocCircle = this.$refs.animationCurrLoactionCirecle;
    this.animConnectedWaves = this.$refs.animationConnectedWaves;

    // resize canvas on window resize
    // (this method should be called each time when main window resizing)
    this.windowResizing();

    this.updateAnimations();
  },

  created: function () {
    window.addEventListener("mousemove", this.windowsmousemove);
  },
  unmounted: function () {
    window.removeEventListener("mousemove", this.windowsmousemove);
  },

  watch: {
    servers() {
      // NOTE! When watching an array, the callback will only trigger when the array is replaced. If you need to trigger on mutation, the 'deep' option must be specified.
      // https://v3-migration.vuejs.org/breaking-changes/watch.html
      this.updateCities();
    },

    connectedLocation() {
      this.updateCities();
      this.updateAnimations();
      this.centerServer(this.connectedLocation);
    },

    selectedServer(newVal, oldVal) {
      this.updateCities();
      this.updateAnimations();
      if (!oldVal || (newVal && oldVal.gateway !== newVal.gateway))
        this.centerServer(this.selectedServer);
    },
    isFastestServer() {
      if (this.isFastestServer !== true && this.isRandomExitServer != true)
        this.centerServer(this.selectedServer);
    },
    isRandomExitServer() {
      if (this.isFastestServer !== true && this.isRandomExitServer != true)
        this.centerServer(this.selectedServer);
    },
    isMinimizedUI() {
      if (!this.isMinimizedUI) this.centerCurrentLocation();
    },
    location() {
      this.updateCities();
      this.updateAnimations();
      setTimeout(() => {
        this.centerCurrentLocation();
      }, 300);
    },
    isLoggedIn() {
      this.centerCurrentLocation();
    },
    async isPaused() {
      this.updateAnimations();
    },
    connectionState() {
      this.updateAnimations();
    },
    isConnecting() {
      // AR-APP-40 - When app starts connecting, ... map should move to position of selected gateway
      if (this.isConnecting)
        setTimeout(() => {
          this.centerServer(this.selectedServer);
        }, 300);
    },
    isConnected() {
      // AR-APP-11 - When VPN status is changed to connected, map should be positioned to the coordinates of selected gateway
      if (this.isConnected) {
        setTimeout(() => {
          this.centerCurrentLocation();
        }, 300);
      }
    },
  },

  methods: {
    mapLoaded() {
      const mapFullW = this.map.naturalWidth;
      const mapFullH = this.map.naturalHeight;
      // scale map
      this.map.style.width = `${mapFullW * this.scale}px`;
      // fit locations container to map scalled size
      this.mapLocationsContainer.style.width = `${mapFullW * this.scale}px`;
      this.mapLocationsContainer.style.height = `${mapFullH * this.scale}px`;

      // To be able to scroll any direction with no border limits,
      // we are creating big scrolling area and map will be placed on center of it
      // So, there will be free space around the map.
      this.mapPos.mapLeftOffset = mapFullW;
      this.mapPos.mapTopOffset = mapFullH;
      this.bigMapArea.style.width = `${mapFullW * 3}px`;
      this.bigMapArea.style.height = `${mapFullH * 3}px`;
      this.mapLocationsContainer.style.left = `${this.mapPos.mapLeftOffset}px`;
      this.mapLocationsContainer.style.top = `${this.mapPos.mapTopOffset}px`;
      this.map.style.left = `${this.mapPos.mapLeftOffset}px`;
      this.map.style.top = `${this.mapPos.mapTopOffset}px`;

      this.updateCities();
      this.centerCurrentLocation(true);
      this.updateAnimations();

      this.isMapLoaded = true;
    },
    windowResizing() {
      // This method should be called each time when main window resizing
      let c = this.canvas;

      if (c == null) return;

      if (c.width === c.clientWidth && c.height === c.clientHeight) return;

      // update canvas size
      c.width = c.clientWidth;
      c.height = c.clientHeight;
    },
    updateColorScheme() {
      let scheme = sender.ColorScheme();
      if (scheme === ColorTheme.system) {
        this.isDarkTheme = IsOsDarkColorScheme();
      } else this.isDarkTheme = scheme === ColorTheme.dark;
    },
    onAccountRenew() {
      sender.shellOpenExternal(`https://www.ivpn.net/account`);
    },
    // ================= CONNECTION ====================
    async disconnect() {
      this.isPopupVisible = false;
      try {
        sender.Disconnect();
      } catch (e) {
        console.error(e);
        sender.showMessageBoxSync({
          type: "error",
          buttons: ["OK"],
          message: `Failed to disconnect: ` + e,
        });
      }
    },

    async connect(location) {
      this.isPopupVisible = false;
      try {
        // check if we can change server (for multihop)
        var settings = this.$store.state.settings;
        if (settings.isMultiHop === true) {
          this.$store.dispatch("settings/isRandomExitServer", false);
          this.$store.dispatch("settings/serverExit", location);
          this.$store.dispatch("settings/serverExitHostId", null);
          await sender.Connect();
        } else {
          this.$store.dispatch("settings/isFastestServer", false);
          this.$store.dispatch("settings/isRandomServer", false);
          this.$store.dispatch("settings/serverEntry", location);
          this.$store.dispatch("settings/serverEntryHostId", null);
          await sender.Connect();
        }
      } catch (e) {
        console.error(e);
        sender.showMessageBoxSync({
          type: "error",
          buttons: ["OK"],
          message: `Failed to connect: ` + e,
        });
      }
    },
    async resume() {
      this.isPopupVisible = false;
      try {
        await sender.ResumeConnection();
      } catch (e) {
        console.error(e);
        sender.showMessageBoxSync({
          type: "error",
          buttons: ["OK"],
          message: `Failed to resume: ` + e,
        });
      }
    },
    // ================= MOUSE ====================
    mouseDown(e) {
      if (this.$store.getters["account/isLoggedIn"] !== true) return;
      this.stopScroll();
      this.isPopupVisible = false;
      this.startMoveX = this.lastMoveX = e.offsetX;
      this.startMoveY = this.lastMoveY = e.offsetY;
      this.moving = true;
    },
    mouseUp() {
      this.moving = false;
    },
    windowsmousemove(e) {
      if (this.moving) {
        // the mouse event out of canvas bounds
        if (e.toElement !== this.canvas) {
          this.mouseMove(e);
        }
      }
    },
    mouseMove(e) {
      if (e.buttons !== 1) {
        // MouseEvent.buttons
        // 0 : No button or un-initialized
        // 1 : Primary button (usually the left button)
        // 2 : Secondary button (usually the right button)
        this.mouseUp();
        return;
      }

      if (this.moving) {
        if (this.map != null) {
          let offsetX = e.offsetX;
          let offsetY = e.offsetY;
          if (e.toElement !== this.canvas) {
            var rect = this.combinedDiv.getBoundingClientRect();
            offsetX = e.clientX - rect.left;
            offsetY = e.clientY - rect.top;
          }

          // MOVING
          // SCROLL HORISONTALLY
          const scrollLeftOffset = this.lastMoveX - offsetX;
          const newScrollLeft = this.combinedDiv.scrollLeft + scrollLeftOffset;

          if (
            // map moving right
            // (minimum visible right map part = 30% of natural width)
            (scrollLeftOffset > 0 &&
              ((newScrollLeft - this.mapPos.mapLeftOffset) / this.scale <
                this.map.naturalWidth * 0.7 ||
                this.map.naturalWidth * this.scale -
                  (newScrollLeft - this.mapPos.mapLeftOffset) >
                  this.canvas.width)) ||
            // map moving left
            // (minimum visible left map part = 30% of natural width)
            (scrollLeftOffset < 0 &&
              (newScrollLeft -
                this.mapPos.mapLeftOffset +
                this.canvas.width -
                this.map.naturalWidth * 0.3 * this.scale >
                0 ||
                newScrollLeft > this.mapPos.mapLeftOffset))
          ) {
            this.combinedDiv.scrollLeft = newScrollLeft;
          }

          // SCROLL VERTICALLY
          const scrollTopOffset = this.lastMoveY - offsetY;
          const newScrollTop = this.combinedDiv.scrollTop + scrollTopOffset;
          if (
            // map moving bottom
            // (minimum visible bottom map part = 2249px+30% of natural height)
            // (2249 pixels at the button of the map is just ocean. No necessary to show it)
            (scrollTopOffset > 0 &&
              ((newScrollTop - this.mapPos.mapTopOffset) / this.scale <
                (this.map.naturalHeight - 2249) * 0.7 ||
                (this.map.naturalHeight - 2249) * this.scale -
                  (newScrollTop - this.mapPos.mapTopOffset) >
                  this.canvas.height)) ||
            // map moving top
            // (minimum visible top map part = 30% of natural height)
            (scrollTopOffset < 0 &&
              (newScrollTop -
                this.mapPos.mapTopOffset +
                this.canvas.height -
                this.map.naturalHeight * 0.3 * this.scale >
                0 ||
                newScrollTop > this.mapPos.mapTopOffset))
          ) {
            this.combinedDiv.scrollTop = newScrollTop;
          }

          this.lastMoveX = offsetX;
          this.lastMoveY = offsetY;
        }
      }
    },

    async locationClicked(location) {
      if (this.$store.getters["account/isLoggedIn"] !== true) return;

      // does selected VPN server location?
      if (location?.gateway) {
        let settings = this.$store.state.settings;
        let conectionState = this.$store.state.vpnState.connectionState;

        if (
          (await CheckAndNotifyInaccessibleServer(
            this.$store,
            settings.isMultiHop,
            location
          )) === true
        ) {
          if (conectionState === VpnStateEnum.DISCONNECTED) {
            if (settings.isMultiHop) {
              this.$store.dispatch("settings/serverExit", location);
              this.$store.dispatch("settings/isRandomExitServer", false);
            } else {
              this.$store.dispatch("settings/serverEntry", location);
              this.$store.dispatch("settings/isRandomServer", false);
              this.$store.dispatch("settings/isFastestServer", false);
            }
          }

          if (settings.connectSelectedMapLocation === true)
            this.connect(location);
        }
      }

      if (this.location == location) {
        // center current location and show popup
        this.centerCurrentLocation();
      } else {
        // center clicked server location
        this.centerServer(location);
      }
    },

    // ================= SCROLLING ====================
    centerCurrentLocation(noAnimation) {
      if (!this.isConnected && this.location != null) {
        this.centerServer(this.location, noAnimation, true);
      } else if (
        !this.isRequestingLocation ||
        (this.combinedDiv.scrollLeft == 0 && this.combinedDiv.scrollTop == 0)
      ) {
        this.centerServer(this.selectedServer, noAnimation);
      }
    },
    centerServer(server, noAnimation, isPopupRequired) {
      if (server == null) return;
      this.windowResizing(); // update canvas size
      let point = getCoordinatesBy(
        server.longitude,
        server.latitude,
        this.scale * this.map.naturalWidth,
        this.scale * this.map.naturalHeight
      );

      let scrollLeft =
        this.mapPos.mapLeftOffset + point.x - this.canvas.width / 2;
      let scrollTop =
        this.mapPos.mapTopOffset + point.y - this.canvas.height / 2;

      if (noAnimation != null && noAnimation) {
        this.stopScroll();
        this.combinedDiv.scrollLeft = scrollLeft;
        this.combinedDiv.scrollTop = scrollTop;

        // Show popup for centered location
        this.showPopupForGeoLocation(server, isPopupRequired);
      } else {
        this.startScroll(scrollLeft, scrollTop, isPopupRequired);
      }
    },
    startScroll(scrollToLeft, scrollTop, isPopupRequired) {
      if (
        this.scrollTo != null &&
        this.scrollTo.left === scrollToLeft &&
        this.scrollTo.top === scrollTop
      )
        return;

      // hide popup (if visible)
      this.isPopupVisible = false;

      this.stopScroll();
      this.scrollTo = {
        left: scrollToLeft,
        top: scrollTop,
        isPopupRequired,
        startTimeMs: performance.now(),
        startLeft: this.combinedDiv.scrollLeft,
        startTop: this.combinedDiv.scrollTop,
        maxDurationMs: 600,
      };

      this.scrollingTick();
    },
    stopScroll() {
      this.scrollTo = null;
    },
    scrollingTick() {
      if (this.scrollTo == null) return;
      const sElement = this.combinedDiv;
      const xOffset = sElement.scrollLeft - this.scrollTo.left;
      const yOffset = sElement.scrollTop - this.scrollTo.top;

      const scrollLeftMax = sElement.scrollWidth - sElement.clientWidth;
      const scrollTopMax = sElement.scrollHeight - sElement.clientHeight;

      const isXScrollFinished =
        Math.abs(xOffset) <= 1 ||
        (sElement.scrollLeft <= 0 && xOffset > 0) ||
        (sElement.scrollLeft >= scrollLeftMax && xOffset < 0);
      const isYScrollFinished =
        Math.abs(yOffset) <= 1 ||
        (sElement.scrollTop <= 0 && yOffset > 0) ||
        (sElement.scrollTop >= scrollTopMax && yOffset < 0);

      if (isXScrollFinished && isYScrollFinished) {
        // Show popup for centered location
        this.showPopup(
          this.scrollTo.left + this.canvas.width / 2,
          this.scrollTo.top + this.canvas.height / 2,
          this.scrollTo.isPopupRequired
        );
        this.stopScroll();
        return;
      }

      let koef =
        (performance.now() - this.scrollTo.startTimeMs) /
        this.scrollTo.maxDurationMs;
      if (koef > 1) koef = 1;

      koef = notLinear(koef);

      sElement.scrollLeft =
        this.scrollTo.startLeft +
        (this.scrollTo.left - this.scrollTo.startLeft) * koef;
      sElement.scrollTop =
        this.scrollTo.startTop +
        (this.scrollTo.top - this.scrollTo.startTop) * koef;

      setTimeout(() => {
        this.scrollingTick();
      }, 1000 / 60);
    },

    // ================= POPUP ====================
    onPopupMouseClick() {
      // AR-APP-20 - Tap / click on the tooltip or any other part of the UI closes the tooltip.
      this.isPopupVisible = false;
    },
    showPopupForGeoLocation(location, isPopupRequired) {
      if (location == null) return;
      let point = this.getLocationVisibleXYCoordinates(location);
      this.showPopup(point.x, point.y, isPopupRequired);
    },
    showPopup(x, y, isPopupRequired) {
      // hide popup (if visible)
      this.isPopupVisible = false;
      if (x == null || y == null) return;

      const mLOffset = this.mapPos.mapLeftOffset;
      const mTOffset = this.mapPos.mapTopOffset;

      // looking for visible locations under map coordinates (x,y)
      x -= mLOffset;
      y -= mTOffset;
      let mapLocation = isUse(this.locationsToDisplay, x, y, 1, 0, 0, 0, 0);

      if (mapLocation == null) return;

      if (isPopupRequired !== true) return;

      // save selected location info: server(or current user location info)
      this.selectedPopupLocation = mapLocation.location;
      // set popup coordinates
      this.popup.style.left =
        mapLocation.x - (this.combinedDiv.scrollLeft - mLOffset) + "px";
      this.popup.style.top =
        mapLocation.y - (this.combinedDiv.scrollTop - mTOffset) + "px";

      // show popup
      this.isPopupVisible = true;
    },
    // ================= ZOOMING ====================
    wheel(e) {
      this.isPopupVisible = false;
      if (e.deltaY > 0) this.zoomIn(e);
      else if (e.deltaY < 0) this.zoomOut(e);
    },
    /*
    async startZooming(expectedScale) {
      if (!expectedScale) return;

      const theThis = this;

      await new Promise(resolve => {
        let zoomTick = function() {
          const isZoomFinished =
            Math.abs(theThis.scale - expectedScale) < 0.0001;
          if (isZoomFinished) {
            resolve();
            return;
          }

          console.log("ZOOM", expectedScale, theThis.scale);
          if (expectedScale > theThis.scale) theThis.zoomOut();
          else if (expectedScale < theThis.scale) theThis.zoomIn();

          setTimeout(async () => {
            zoomTick();
          }, 1000 / 60);
        };
        zoomTick();
      });
    },*/

    zoomIn(e) {
      const step = 0.025;
      const minScale = 0.09;
      let newScale = this.scale - step;

      if (this.scale <= minScale) return;
      if (newScale < minScale) newScale = minScale;

      this.updateScale(newScale, e);
    },
    zoomOut(e) {
      const step = 0.025;
      const maxScale = 1.0;
      let newScale = this.scale + step;

      if (this.scale >= maxScale) return;
      if (newScale > maxScale) newScale = maxScale;

      this.updateScale(newScale, e);
    },

    updateScale(newScale, zoomPoint) {
      const scaleDiff = newScale - this.scale;

      if (Math.abs(scaleDiff) < 0.001) return;
      const l = this.combinedDiv.scrollLeft;
      const t = this.combinedDiv.scrollTop;

      if (zoomPoint && zoomPoint.srcElement != this.canvas) {
        // if event come from non-canvas element (mouse is over location point\text)
        // - necessary to recalculate mouse position accourding canvas area
        zoomPoint = {
          offsetX: zoomPoint.clientX - config.MinimizedUIWidth,
          offsetY: zoomPoint.clientY,
        };
      }
      // save geo coordinates of center point (to be able to center same point after scalling)
      else if (zoomPoint == null) {
        zoomPoint = {
          offsetX: this.canvas.width / 2,
          offsetY: this.canvas.height / 2,
        };
      }

      const mapFullW = this.map.naturalWidth;
      const mapFullH = this.map.naturalHeight;
      const mLOffset = this.mapPos.mapLeftOffset;
      const mTOffset = this.mapPos.mapTopOffset;

      const centerCoord = getPosFromCoordinates(
        l + zoomPoint.offsetX - mLOffset,
        t + zoomPoint.offsetY - mTOffset,
        this.scale * mapFullW,
        this.scale * mapFullH
      );

      // change scale
      this.scale = newScale;
      this.map.style.width = `${mapFullW * this.scale}px`;
      this.mapLocationsContainer.style.width = `${mapFullW * this.scale}px`;
      this.mapLocationsContainer.style.height = `${mapFullH * this.scale}px`;

      // keep scroll positions
      let newPoint = getCoordinatesBy(
        centerCoord.longitude,
        centerCoord.latitude,
        this.scale * mapFullW,
        this.scale * mapFullH
      );

      this.combinedDiv.scrollLeft = mLOffset + newPoint.x - zoomPoint.offsetX;
      this.combinedDiv.scrollTop = mTOffset + newPoint.y - zoomPoint.offsetY;

      // redraw locations
      this.updateCities();

      // necessary to keep correct circle locations
      this.animateCurentLocation();
      this.animateSelectedServer();
    },

    // ================= DRAWING ====================
    getLocationXYCoordinates(location) {
      if (location == null || this.map == null || this.map.naturalWidth == null)
        return null;
      let point = getCoordinatesBy(
        location.longitude,
        location.latitude,
        this.scale * this.map.naturalWidth,
        this.scale * this.map.naturalHeight
      );
      return point;
    },

    getLocationVisibleXYCoordinates(location) {
      if (location == null) return null;
      let point = getCoordinatesBy(
        location.longitude,
        location.latitude,
        this.scale * this.map.naturalWidth,
        this.scale * this.map.naturalHeight
      );
      if (point == null) return null;
      point.x = point.x - this.combinedDiv.scrollLeft;
      point.y = point.y - this.combinedDiv.scrollTop;
      return point;
    },

    updateCities() {
      let cities = [];
      const PointRadius = 3;
      const PointRadiusExitServer = 6;
      const settings = this.$store.state.settings;

      // Selected servers and current location has highest priority to draw
      let city = null;
      // current location
      if (this.location != null) {
        city = this.createCity(this.location, cities, PointRadius);
        if (city != null) cities.push(city);
      }

      // show exit- and entry- servers
      city = this.createCity(
        this.connectedLocation,
        cities,
        PointRadiusExitServer
      );
      if (city != null) cities.push(city);
      // if MultiHop - just ensure that entry- server vill be visible
      if (settings.isMultiHop) {
        let city = this.createCity(settings.serverEntry, cities, PointRadius);
        if (city != null) cities.push(city);
      }

      let skippedCities = [];
      // all the rest locations
      this.servers.forEach((s) => {
        city = this.createCity(s, cities, PointRadius);
        if (city == null) {
          skippedCities.push(s);
          return;
        }
        cities.push(city);
      });

      // if there is no space to show location -> trying to show at least points (without name)
      const doNotShowName = true;
      skippedCities.forEach((s) => {
        city = this.createCity(s, cities, PointRadius, doNotShowName);
        if (city == null) return;
        cities.push(city);
      });
      this.locationsToDisplay = cities;
    },

    createCity(location, locations, pointRadius, doNotShowName) {
      let point = this.getLocationXYCoordinates(location);
      if (point == null) return;
      let x = point.x;
      let y = point.y;

      let textWidth = 0;
      let textHeight = 0;
      if (doNotShowName == null || doNotShowName == false) {
        this.hiddenTestTextMeter.innerText = location.city;
        textWidth = this.hiddenTestTextMeter.clientWidth;
        textHeight = this.hiddenTestTextMeter.clientHeight;
      }

      let textPos = calcTextLocation(
        x,
        y,
        pointRadius,
        textWidth,
        textHeight,
        locations
      );
      if (textPos == null) {
        //console.log(`Skipped '${location.city}' due to overlapping`);
        return;
      }
      return {
        location,
        x,
        y,
        left: textPos.textX,
        top: textPos.textY,
        pointRadius,
        width: textWidth,
        height: textHeight,
      };
    },

    // ================= ANIMATIONS ================
    updateAnimations() {
      this.animateCurentLocation();
      this.animateSelectedServer();
    },

    animateCurentLocation() {
      // Current location
      if (this.location == null || !this.isDisconnected) {
        // Location not known
        // remove 'grow' style
        this.animCurrLocCircle.classList.remove("mapLocationCircleGrow");
        // add 'shrink' style
        if (
          !this.animCurrLocCircle.classList.contains("mapLocationCircleShrink")
        ) {
          this.animCurrLocCircle.classList.add("mapLocationCircleShrink");
        }
      } else {
        const point = this.getLocationXYCoordinates(this.location);
        if (point != null) {
          // set coordinates
          this.animCurrLocCircle.style.left = `${point.x}px`;
          this.animCurrLocCircle.style.top = `${point.y}px`;

          // remove 'shrink' style
          this.animCurrLocCircle.classList.remove("mapLocationCircleShrink");

          // add 'grow' style
          if (
            !this.animCurrLocCircle.classList.contains("mapLocationCircleGrow")
          ) {
            this.animCurrLocCircle.classList.add("mapLocationCircleGrow");
          }
        }
      }
    },
    animateSelectedServer() {
      function setDisconnectedObj(theThis) {
        const el1 = theThis.animSelCircles[0];
        const el2 = theThis.animSelCircles[1];

        // change color to disconnected
        el1.classList.remove("mapLocationCircleConnected");
        if (!el1.classList.contains("mapLocationCircleDisonnected"))
          el1.classList.add("mapLocationCircleDisonnected");

        // shrink circle
        el1.classList.remove("mapLocationCircleGrow");
        if (!el1.classList.contains("mapLocationCircleShrink"))
          el1.classList.add("mapLocationCircleShrink");

        // put object to the and of 'animSelCircles' array
        // The last element must be allways shrinked, first - ready to grow
        theThis.animSelCircles[0] = el2;
        theThis.animSelCircles[1] = el1;
      }

      let obj = this.animSelCircles[0];
      if (this.isInProgress || this.isConnected) {
        //if (this.isConnecting || this.isConnected) {
        const point = this.getLocationXYCoordinates(this.connectedLocation);
        if (point != null) {
          if (
            obj.IVPNLocationObj != null &&
            obj.IVPNLocationObj.city != this.connectedLocation.city
          ) {
            setDisconnectedObj(this);
            obj = this.animSelCircles[0];
          }

          // set coordinates for selected server
          obj.style.left = `${point.x}px`;
          obj.style.top = `${point.y}px`;
          obj.IVPNLocationObj = this.connectedLocation;

          // set coordinates for 'connected waves' animation object
          this.animConnectedWaves.style.left = `${point.x}px`;
          this.animConnectedWaves.style.top = `${point.y}px`;
          this.animConnectedWaves.IVPNLocationObj = this.connectedLocation;

          // we are connected or connecting
          // Remove 'shrink' class
          obj.classList.remove("mapLocationCircleShrink");
          // Add 'grow' class
          if (!obj.classList.contains("mapLocationCircleGrow"))
            obj.classList.add("mapLocationCircleGrow");

          if (this.isInProgress) {
            // change color to 'disconnected'
            obj.classList.remove("mapLocationCircleConnected");

            if (!obj.classList.contains("mapLocationCircleDisonnected"))
              obj.classList.add("mapLocationCircleDisonnected");
          }

          if (this.isConnected) {
            // change color to 'connected'
            obj.classList.remove("mapLocationCircleDisonnected");

            if (!obj.classList.contains("mapLocationCircleConnected"))
              obj.classList.add("mapLocationCircleConnected");
          }
        }
      } else setDisconnectedObj(this);
    },
  },
};

function calcTextLocation(
  x,
  y,
  pointRadius,
  textWidth,
  textHeight,
  drawedCities
) {
  const space = 3;
  const width = textWidth;
  const height = textHeight;

  // up (above the point)
  let left = x - width / 2;
  let top = y - (pointRadius + space + height);
  if (!isUse(drawedCities, x, y, pointRadius, left, top, width, height) == true)
    return { textX: left, textY: top };

  // right
  left = x + space + pointRadius;
  top = y - height / 2;
  if (!isUse(drawedCities, x, y, pointRadius, left, top, width, height) == true)
    return { textX: left, textY: top };

  // buttom (under the point)
  left = x - width / 2;
  top = y + (pointRadius + space);
  if (!isUse(drawedCities, x, y, pointRadius, left, top, width, height) == true)
    return { textX: left, textY: top };

  // left
  left = x - pointRadius - space - width;
  top = y - height / 2;
  if (!isUse(drawedCities, x, y, pointRadius, left, top, width, height) == true)
    return { textX: left, textY: top };

  // up + right
  left = x - pointRadius;
  top = y - (pointRadius + space + height);
  if (!isUse(drawedCities, x, y, pointRadius, left, top, width, height) == true)
    return { textX: left, textY: top };

  // up + left
  left = x - width + pointRadius;
  top = y - (pointRadius + space + height);
  if (!isUse(drawedCities, x, y, pointRadius, left, top, width, height) == true)
    return { textX: left, textY: top };

  // buttom + right
  left = x - pointRadius;
  top = y + (pointRadius + space);
  if (!isUse(drawedCities, x, y, pointRadius, left, top, width, height) == true)
    return { textX: left, textY: top };

  // buttom + left
  left = x - width + pointRadius;
  top = y + (pointRadius + space);
  if (!isUse(drawedCities, x, y, pointRadius, left, top, width, height) == true)
    return { textX: left, textY: top };

  return null;
}

function isUse(drawedCities, x, y, pointRadius, left, top, width, height) {
  let isRectInside = function (r1, r2) {
    return (
      r1.left + r1.width >= r2.left &&
      r1.left <= r2.left + r2.width &&
      r1.top <= r2.top + r2.height &&
      r1.top + r1.height >= r2.top
    );
  };

  let r1 = { left, top, width, height };
  for (let i = 0; i < drawedCities.length; i++) {
    let r2 = drawedCities[i];
    let p1 = {
      left: x - pointRadius,
      top: y - pointRadius,
      width: pointRadius * 2,
      height: pointRadius * 2,
    };
    let p2 = {
      left: r2.x - pointRadius,
      top: r2.y - pointRadius,
      width: r2.pointRadius * 2,
      height: r2.pointRadius * 2,
    };

    if (isRectInside(r1, r2)) return drawedCities[i];
    if (isRectInside(p1, r2)) return drawedCities[i];
    if (isRectInside(p1, p2)) return drawedCities[i];
    if (isRectInside(r1, p2)) return drawedCities[i];
  }
  return null;
}
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
//$shadow: 0px 4px 24px rgba(37, 51, 72, 0.25);

$shadow: 0px 3px 12px rgba(var(--shadow-color-rgb), var(--shadow-opacity));

$popup-background: var(--background-color);

@import "@/components/scss/constants";

#main {
  position: relative;
  width: 100%;
  height: 100%;
}

.mapcontainer {
  height: inherit;
  width: inherit;

  display: block;
  overflow-x: hidden;
  overflow-y: hidden;
}

.canvas {
  position: absolute;
  z-index: 2;
  height: inherit;
  width: inherit;
}

.canvasTop {
  @extend .canvas;
  z-index: 5;
  pointer-events: none;
}

.blured {
  filter: blur(7px);
  backdrop-filter: blur(5px);
}
.bigMapArea {
  position: relative;
  background: var(--map-background-color);
}
.map {
  position: relative;
  z-index: 1;

  user-select: none;
}

.buttonsPanelBase {
  position: absolute;
  z-index: 4;
}

.buttonsPanelTopBase {
  @extend .buttonsPanelBase;
  margin-top: 18px;
}

.buttonsPanelTopRight {
  @extend .buttonsPanelTopBase;
  left: 100%;
  margin-left: -166px;
}

.buttonsPanelTopRightNoFrameWindow {
  margin-left: -250px;
}

.buttonsPanelTopLeft {
  @extend .buttonsPanelTopBase;
  left: 28px;
}

.bottomButtonsPanel {
  @extend .buttonsPanelBase;
  left: 100%;
  margin-left: -54px;

  top: 100%;
  margin-top: -60px;
}

.bottomPanel {
  @extend .buttonsPanelBase;
  bottom: 0;
  margin-bottom: 32px;
  margin-left: 32px;
  width: calc(100% - 64px);

  pointer-events: none;
}

.geolocationInfoPanel {
  //pointer-events: auto;
  padding: 10px;

  background: rgba(var(--background-color-rgb), 0.6);

  backdrop-filter: blur(4px);
  border-radius: 4px;

  min-height: 76px;
  min-width: 176px;

  display: flex;
  align-items: center;
}

// ============== account expiration =================
div.upgradeBase {
  min-height: 40px;
  border-radius: 8px;

  display: flex;
  align-items: center;

  font-size: 12px;
  line-height: 14px;
  letter-spacing: -0.4px;

  margin-top: 12px;
  padding-left: 12px;
  padding-right: 12px;
}
div.upgradeBase button {
  font-size: 12px;
  line-height: 14px;
  letter-spacing: -0.4px;
  pointer-events: auto;
  //cursor: pointer;
}
div.accountWillExpire {
  @extend .upgradeBase;
  background: #f3e2a3;
  color: #776832;
}
div.accountWillExpire button {
  color: #776832;
}
div.trialWillExpire {
  @extend .upgradeBase;
  background: #b2d5fb;
  color: #3e6894;
}
div.trialWillExpire button {
  color: #3e6894;
}

// ============== settings =================

.settingsBtnMarginLeft {
  margin-left: 24px;
}
.settingsBtn {
  float: right;

  width: 32px;
  height: 32px;

  padding: 0px;
  border: none;
  border-radius: 50%;
  background-color: var(--background-color);
  outline-width: 0;
  cursor: pointer;

  box-shadow: $shadow;

  // centering content
  display: flex;
  justify-content: center;
  align-items: center;
}

.settingsBtn:hover {
  background-color: var(--background-color-alternate);
}

// ============== POPUP =================
// Popup container - can be anything you want
.popup {
  position: absolute;
  z-index: 5;
  user-select: none;
}

// The actual popup
.popup .popuptext {
  visibility: hidden;
  background-color: $popup-background;
  text-align: center;
  border-radius: 8px;
  position: absolute;

  margin-top: 24px; // 15 arrow + 9

  min-width: 270px;
  max-width: 270px;
  margin-left: -149px; // 270/2 + padding(14)

  padding: 14px;
  box-shadow: $shadow;
}

// Popup arrow
.popup .popuptext::after {
  content: "";
  position: absolute;
  top: -30px;
  margin-left: -15px;
  border-width: 15px;
  border-style: solid;
  border-color: transparent transparent $popup-background transparent;
}

// Toggle this class - hide and show the popup
.popup .show {
  visibility: visible;
  animation: fadeIn 0.5s;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}
// ========== LOCATIONS ===========
.mapLocationsContainer {
  position: absolute;
  z-index: 3;
  pointer-events: none;
}

.mapLocationElement {
  position: absolute;
  cursor: pointer;
  pointer-events: all;
  white-space: nowrap;
}

.mapLocationName {
  @extend .mapLocationElement;
  font-size: 10px;
  line-height: 12px;
  display: inline-block;
  letter-spacing: -0.3px;

  color: var(--map-text-color);
  text-shadow: -1px 1px 0 var(--background-color),
    1px 1px 0 var(--background-color), 1px -1px 0 var(--background-color),
    -1px -1px 0 var(--background-color);
}

.mapLocationNameCurrent {
  color: #ff6258;
}

.mapLocationNameConnected {
  color: #449cf8;
}

.mapLocationPoint {
  @extend .mapLocationElement;
  background-color: var(--map-point-color);
  border-radius: 100%;
  display: inline-block;
}

.mapLocationPointCurrent {
  background: #ff6258;
}

.mapLocationPointConnected {
  background: #449cf8;
}

// ========== ANIMATIONS ===========
.mapLocationCircle {
  position: absolute;
  pointer-events: none;
  white-space: nowrap;
  opacity: 0;

  transform: scale(0);

  border-radius: 100%;
  display: inline-block;

  transition: transform 0.7s, opacity 0.7s, background 0.5s;
}

.mapLocationCircleSelectedSvr {
  @extend .mapLocationCircle;
  margin-left: -48px;
  margin-top: -48px;
  width: 96px;
  height: 96px;
  background: #6b6b6b;
}

.mapLocationCircleConnectedWaves {
  @extend .mapLocationCircleSelectedSvr;
  @extend .mapLocationCircleConnected;
}
.mapLocationCircleConnectedWavesRunning {
  animation: growWave 5s infinite;
}

.mapLocationCircleCurrLocation {
  @extend .mapLocationCircle;
  margin-left: -48px;
  margin-top: -48px;
  width: 96px;
  height: 96px;
  background: #ff6258;
}

.mapLocationCircleDisonnected {
  background: #6b6b6b;
}
.mapLocationCircleConnected {
  background: #449cf8;
}

.mapLocationCircleShrink {
  opacity: 0;
  transform: scale(0);
}

.mapLocationCircleGrow {
  opacity: 0.5;
  transform: scale(1);
}

@keyframes growWave {
  0% {
    opacity: 0.5;
    transform: scale(0);
  }
  60% {
    // delay between iterations=5s, animation=3s (3/5=0.6)
    opacity: 0;
    transform: scale(1);
  }
  100% {
    opacity: 0;
    transform: scale(1);
  }
}
</style>
