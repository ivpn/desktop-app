<template>
  <div id="main">
    <!-- Popup -->
    <div class="popup">
      <div
        ref="popup"
        class="popuptext"
        v-bind:class="{
          show: isPopupVisible && isInProgress == false && isBlured !== 'true'
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

    <!-- Buttons panel -->
    <div class="buttonsPanel" v-if="isBlured !== 'true'">
      <button class="settingsBtn settingsBtnMarginLeft" v-on:click="onSettings">
        <img src="@/assets/settings.svg" />
      </button>

      <button class="settingsBtn" v-on:click="onAccountSettings">
        <img src="@/assets/user.svg" />
      </button>
    </div>

    <!-- Bottom panel -->
    <div class="bottomButtonsPanel" v-if="isBlured !== 'true'">
      <button class="settingsBtn" v-on:click="centerCurrentLocation(false)">
        <img src="@/assets/crosshair.svg" />
      </button>
    </div>

    <!-- Map -->
    <div class="mapcontainer" ref="combined">
      <canvas
        class="canvas"
        ref="canvas"
        v-bind:class="{ blured: isBlured === 'true' }"
        @mousedown="mouseDown"
        @mouseup="mouseUp"
        @mouseleave="mouseUp"
        @mousemove="mouseMove"
        @wheel="wheel"
      >
      </canvas>

      <!-- Top-located canvas to be able to blure map -->
      <canvas
        class="canvasTop"
        v-bind:class="{ blured: isBlured === 'true' }"
      ></canvas>

      <div style="position: relative ">
        <img
          ref="map"
          class="map"
          src="@/assets/world_map_light.svg"
          @load="mapLoaded"
        />

        <!-- Hidden element to calculate styled text size-->
        <div
          ref="hiddenTestTextMeter"
          class="mapLocationName"
          style="opacity: 0, pointer-events: none; z-index: -1;"
        ></div>

        <div class="mapLocationsContainer" ref="mapLocationsContainer">
          <!-- Location point -->
          <div
            class="mapLocationPoint"
            v-for="l of locationsToDisplay"
            v-on:click="locationClicked(l.location)"
            v-bind:key="'point_' + l.location.city"
            v-bind:class="{
              mapLocationPointCurrent: l.location === location,
              mapLocationPointConnected:
                l.location === selectedServer && isConnected
            }"
            :style="{
              left: l.x - l.pointRadius + 'px',
              top: l.y - l.pointRadius + 'px',
              height: l.pointRadius * 2 + 'px',
              width: l.pointRadius * 2 + 'px'
            }"
          ></div>
          <!-- Location name -->
          <div
            class="mapLocationName"
            v-for="l of locationsToDisplay"
            v-on:click="locationClicked(l.location)"
            v-bind:key="'name_' + l.location.city"
            v-bind:class="{
              mapLocationNameCurrent: l.location === location,
              mapLocationNameConnected:
                l.location === selectedServer && isConnected
            }"
            :style="{ left: l.left + 'px', top: l.top + 'px' }"
          >
            {{ l.width > 0 ? l.location.city : "" }}
          </div>

          <!-- Animation elements -->
          <div
            ref="animationCurrLoactionCirecle"
            class="mapLocationCircleCurrLocation"
          ></div>

          <div
            ref="animationSelectedCirecle1"
            class="mapLocationCircleSelectedSvr"
          ></div>
          <div
            ref="animationSelectedCirecle2"
            class="mapLocationCircleSelectedSvr"
          ></div>

          <div
            ref="animationConnectedWaves"
            class="mapLocationCircleConnectedWaves"
            v-bind:class="{
              mapLocationCircleConnectedWavesRunning: isConnected
            }"
          ></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
const { dialog, getCurrentWindow } = require("electron").remote;

import { VpnStateEnum, PauseStateEnum } from "@/store/types";

import sender from "@/ipc/renderer-sender";
import popupControl from "@/components/controls/control-map-popup.vue";
import {
  notLinear,
  getPosFromCoordinates,
  getCoordinatesBy
} from "@/helpers/helpers";

export default {
  components: {
    popupControl
  },
  props: {
    isBlured: String,
    onAccountSettings: Function,
    onSettings: Function
  },

  data: () => ({
    selectedPopupLocation: null,
    isPopupVisible: false,

    canvas: null,

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

    scale: 0.41,

    scrollTo: {
      left: -1,
      top: -1,
      isPopupRequired: false,
      startTimeMs: null,
      startLeft: -1,
      startTop: -1,
      maxDurationMs: 0
    },

    moving: false,
    startMoveX: 0,
    startMoveY: 0,
    lastMoveX: 0,
    lastMoveY: 0,

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
    ]
  }),

  computed: {
    servers: function() {
      return this.$store.getters["vpnState/activeServers"];
    },
    entryServer: function() {
      return this.$store.state.settings.serverEntry;
    },
    selectedServer: function() {
      if (this.$store.state.settings.isMultiHop)
        return this.$store.state.settings.serverExit;
      return this.$store.state.settings.serverEntry;
    },

    location: function() {
      return this.$store.state.location;
    },

    isPaused: function() {
      return this.$store.state.vpnState.pauseState === PauseStateEnum.Paused;
    },

    currentPosition: function() {
      if (this.isPaused) return this.location;
      return this.isInProgress || this.isConnected
        ? this.selectedServer
        : this.location;
    },

    currentPositionAbsoluteCoordinates: function() {
      const cp = this.currentPosition;
      if (cp == null) return { x: 0, y: 0 };
      const point = this.getLocationXYCoordinates(cp);
      if (point == null) return { x: 0, y: 0 };
      return point;
    },

    currentPositionCircleRadius: function() {
      const cp = this.currentPosition;
      if (cp == null) return 0;
      if (cp === this.selectedServer) return 44;
      return 32;
    },

    // needed for watcher
    isLoggedIn: function() {
      return this.$store.getters["account/isLoggedIn"];
    },
    // needed for watcher
    connectionState: function() {
      return this.$store.state.vpnState.connectionState;
    },
    isConnecting: function() {
      return (
        this.$store.state.vpnState.connectionState === VpnStateEnum.CONNECTING
      );
    },
    isConnected: function() {
      if (this.isPaused) return false;
      return (
        this.$store.state.vpnState.connectionState === VpnStateEnum.CONNECTED
      );
    },
    isDisconnected: function() {
      if (this.isPaused) return true;
      return (
        this.$store.state.vpnState.connectionState === VpnStateEnum.DISCONNECTED
      );
    },
    isInProgress: function() {
      return !this.isConnected && !this.isDisconnected;
    }
  },

  mounted() {
    this.canvas = this.$refs.canvas;

    this.popup = this.$refs.popup;
    this.map = this.$refs.map;
    this.combinedDiv = this.$refs.combined;
    this.mapLocationsContainer = this.$refs.mapLocationsContainer;
    this.hiddenTestTextMeter = this.$refs.hiddenTestTextMeter;

    this.animSelCircles.push(this.$refs.animationSelectedCirecle1);
    this.animSelCircles.push(this.$refs.animationSelectedCirecle2);
    this.animCurrLocCircle = this.$refs.animationCurrLoactionCirecle;
    this.animConnectedWaves = this.$refs.animationConnectedWaves;

    // resize canvas on window resize
    let currentWindow = getCurrentWindow();
    currentWindow.on("resize", this.windowResizing);
    this.windowResizing();

    this.updateAnimations();
  },

  watch: {
    selectedServer(oldVal, newVal) {
      this.updateCities();
      this.updateAnimations();
      if (oldVal !== newVal) this.centerServer(this.selectedServer);
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
    async isDisconnected() {
      try {
        // AR-APP-12 - When VPN status is changed to disconnected, app should call the geolocation API to get user's location, and the location on the map should be updated.
        if (this.isDisconnected) await sender.GeoLookup();
      } catch (e) {
        console.error(e);
      }
    }
  },

  methods: {
    mapLoaded() {
      this.map.style.width = `${this.map.naturalWidth * this.scale}px`;

      this.mapLocationsContainer.style.width = `${this.map.naturalWidth *
        this.scale}px`;
      this.mapLocationsContainer.style.height = `${this.map.naturalHeight *
        this.scale}px`;
      this.mapLocationsContainer.style.left = "0px";
      this.mapLocationsContainer.style.top = "0px";

      this.updateCities();
      this.centerCurrentLocation(true);
      this.updateAnimations();
    },
    windowResizing() {
      let c = this.canvas;

      if (c == null) return;

      if (c.width === c.clientWidth && c.height === c.clientHeight) return;

      // update canvas size
      c.width = c.clientWidth;
      c.height = c.clientHeight;
    },
    // ================= CONNECTION ====================
    async disconnect() {
      this.isPopupVisible = false;
      try {
        sender.Disconnect();
      } catch (e) {
        console.error(e);
        dialog.showMessageBoxSync(getCurrentWindow(), {
          type: "error",
          buttons: ["OK"],
          message: `Failed to disconnect: ` + e
        });
      }
    },

    async connect(location) {
      this.isPopupVisible = false;
      try {
        if (this.$store.state.settings.isMultiHop)
          await sender.Connect(null, location);
        else await sender.Connect(location, null);
      } catch (e) {
        console.error(e);
        dialog.showMessageBoxSync(getCurrentWindow(), {
          type: "error",
          buttons: ["OK"],
          message: `Failed to connect: ` + e
        });
      }
    },
    async resume() {
      this.isPopupVisible = false;
      try {
        await sender.ResumeConnection();
      } catch (e) {
        console.error(e);
        dialog.showMessageBoxSync(getCurrentWindow(), {
          type: "error",
          buttons: ["OK"],
          message: `Failed to resume: ` + e
        });
      }
    },
    // ================= MOUSE ====================
    mouseDown(e) {
      this.stopScroll();
      this.isPopupVisible = false;
      this.startMoveX = this.lastMoveX = e.offsetX;
      this.startMoveY = this.lastMoveY = e.offsetY;
      this.moving = true;
    },
    mouseUp() {
      this.moving = false;
    },
    mouseMove(e) {
      if (this.moving) {
        if (this.map != null) {
          // MOVING
          this.combinedDiv.scrollLeft -= e.offsetX - this.lastMoveX;
          this.combinedDiv.scrollTop -= e.offsetY - this.lastMoveY;

          this.lastMoveX = e.offsetX;
          this.lastMoveY = e.offsetY;
        }
      }
    },

    locationClicked(location) {
      if (
        location != null &&
        location.gateway != null &&
        this.$store.state.vpnState.connectionState === VpnStateEnum.DISCONNECTED
      ) {
        if (this.$store.state.settings.isMultiHop) {
          if (
            location.country_code !==
            this.$store.state.settings.serverEntry.country_code
          )
            this.$store.dispatch("settings/serverExit", location);
        } else this.$store.dispatch("settings/serverEntry", location);

        this.$store.dispatch("settings/isFastestServer", false);
        this.$store.dispatch("settings/isRandomServer", false);
      }

      const isPopupRequired = true;
      const noAnimation = false;
      this.centerServer(location, noAnimation, isPopupRequired);
    },

    // ================= SCROLLING ====================
    centerCurrentLocation(noAnimation) {
      if (!this.isConnected && this.location != null)
        this.centerServer(this.location, noAnimation);
      else this.centerServer(this.selectedServer, noAnimation);
    },
    centerServer(server, noAnimation, isPopupRequired) {
      if (server == null) return;
      let point = getCoordinatesBy(
        server.longitude,
        server.latitude,
        this.scale * this.map.naturalWidth,
        this.scale * this.map.naturalHeight
      );

      let scrollLeft = point.x - this.canvas.width / 2;
      let scrollTop = point.y - this.canvas.height / 2;

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
        maxDurationMs: 600
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

      if (
        (Math.abs(xOffset) <= 1 ||
          (sElement.scrollLeft <= 0 && xOffset > 0) ||
          (sElement.scrollLeft >= scrollLeftMax && xOffset < 0)) &&
        (Math.abs(yOffset) <= 1 ||
          (sElement.scrollTop <= 0 && yOffset > 0) ||
          (sElement.scrollTop >= scrollTopMax && yOffset < 0))
      ) {
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
      // looking for visible locations under map coordinates (x,y)
      let mapLocation = isUse(this.locationsToDisplay, x, y, 1, 0, 0, 0, 0);

      if (mapLocation == null) return;
      // do not show 'disconnect' popup
      if (
        isPopupRequired !== true &&
        !this.isCanShowPopupForLocation(mapLocation.location)
      ) {
        return;
      }

      // save selected location info: server(or current user location info)
      this.selectedPopupLocation = mapLocation.location;
      // set popup coordinates
      this.popup.style.left =
        mapLocation.x - this.combinedDiv.scrollLeft + "px";
      this.popup.style.top = mapLocation.y - this.combinedDiv.scrollTop + "px";

      // show popup
      this.isPopupVisible = true;
    },
    isCanShowPopupForLocation(location) {
      if (this.isConnected && location === this.selectedServer) return false;
      return true;
    },
    // ================= ZOOMING ====================
    wheel(e) {
      this.isPopupVisible = false;
      if (e.deltaY > 0) this.zoomIn(e);
      else this.zoomOut(e);
    },
    zoomIn(e) {
      if (this.scale <= 0.3) return;
      this.updateScale(this.scale - 0.025, e);
    },
    zoomOut(e) {
      if (this.scale >= 1) return;
      this.updateScale(this.scale + 0.025, e);
    },

    updateScale(newScale, zoomPoint) {
      const scaleDiff = newScale - this.scale;

      if (Math.abs(scaleDiff) < 0.001) return;
      const l = this.combinedDiv.scrollLeft;
      const t = this.combinedDiv.scrollTop;

      // save geo coordinates of center point (to be able to center same point after scalling)
      if (zoomPoint == null) {
        zoomPoint = {
          offsetX: this.canvas.width / 2,
          offsetY: this.canvas.height / 2
        };
      }
      const centerCoord = getPosFromCoordinates(
        l + zoomPoint.offsetX,
        t + zoomPoint.offsetY,
        this.scale * this.map.naturalWidth,
        this.scale * this.map.naturalHeight
      );

      // change scale
      this.scale = newScale;
      this.map.style.width = `${this.map.naturalWidth * this.scale}px`;

      this.mapLocationsContainer.style.width = `${this.map.naturalWidth *
        this.scale}px`;
      this.mapLocationsContainer.style.height = `${this.map.naturalHeight *
        this.scale}px`;

      // keep scroll positions
      let newPoint = getCoordinatesBy(
        centerCoord.longitude,
        centerCoord.latitude,
        this.scale * this.map.naturalWidth,
        this.scale * this.map.naturalHeight
      );
      this.combinedDiv.scrollLeft = newPoint.x - zoomPoint.offsetX;
      this.combinedDiv.scrollTop = newPoint.y - zoomPoint.offsetY;

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

      // Selected servers and current location has highest priority to draw
      let city = null;
      // current location
      if (this.location != null) {
        city = this.createCity(this.location, cities, PointRadius);
        if (city != null) cities.push(city);
      }
      // entry- exit- servers
      const settings = this.$store.state.settings;
      if (settings.isMultiHop) {
        city = this.createCity(
          settings.serverExit,
          cities,
          PointRadiusExitServer
        );
        if (city != null) cities.push(city);

        city = this.createCity(settings.serverEntry, cities, PointRadius);
        if (city != null) cities.push(city);
      } else {
        city = this.createCity(
          settings.serverEntry,
          cities,
          PointRadiusExitServer
        );
        if (city != null) cities.push(city);
      }

      let skippedCities = [];
      // all the rest locations
      this.servers.forEach(s => {
        city = this.createCity(s, cities, PointRadius);
        if (city == null) {
          skippedCities.push(s);
          return;
        }
        cities.push(city);
      });

      // if there is no space to show location -> trying to show at least points (without name)
      const doNotShowName = true;
      skippedCities.forEach(s => {
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
        this.hiddenTestTextMeter.innerHTML = location.city;
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
        height: textHeight
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
        const point = this.getLocationXYCoordinates(this.selectedServer);
        if (point != null) {
          if (
            obj.IVPNLocationObj != null &&
            obj.IVPNLocationObj.city != this.selectedServer.city
          ) {
            setDisconnectedObj(this);
            obj = this.animSelCircles[0];
          }

          // set coordinates for selected server
          obj.style.left = `${point.x}px`;
          obj.style.top = `${point.y}px`;
          obj.IVPNLocationObj = this.selectedServer;

          // set coordinates for 'connected waves' animation object
          this.animConnectedWaves.style.left = `${point.x}px`;
          this.animConnectedWaves.style.top = `${point.y}px`;
          this.animConnectedWaves.IVPNLocationObj = this.selectedServer;

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
    }
  }
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
  let isRectInside = function(r1, r2) {
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
      height: pointRadius * 2
    };
    let p2 = {
      left: r2.x - pointRadius,
      top: r2.y - pointRadius,
      width: r2.pointRadius * 2,
      height: r2.pointRadius * 2
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
$shadow: 0px 4px 24px rgba(37, 51, 72, 0.25);
$popup-background: white;

@import "@/components/scss/constants";
$mapBackground: #cbd2d3;

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

  background: $mapBackground;
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

.map {
  position: relative;
  z-index: 1;

  user-select: none;
  background: $mapBackground;
}

.buttonsPanel {
  left: 100%;
  margin-left: -110px;

  position: absolute;
  z-index: 4;

  margin-top: 18px;
}
.bottomButtonsPanel {
  left: 100%;
  margin-left: -54px;

  top: 100%;
  margin-top: -60px;

  position: absolute;
  z-index: 4;
}

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
  background-color: #ffffff;
  outline-width: 0;
  cursor: pointer;

  box-shadow: $shadow;

  // centering content
  display: flex;
  justify-content: center;
  align-items: center;
}

.settingsBtn:hover {
  background-color: #f0f0f0;
}

// ============== POPUP =================
// Popup container - can be anything you want
.popup {
  position: absolute;
  z-index: 4;
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
  color: #6b6b6b;
  text-shadow: -1px 1px 0 #ffffff, 1px 1px 0 #ffffff, 1px -1px 0 #ffffff,
    -1px -1px 0 #ffffff;
}

.mapLocationNameCurrent {
  color: #ff6258;
}

.mapLocationNameConnected {
  color: #449cf8;
}

.mapLocationPoint {
  @extend .mapLocationElement;
  background-color: #6b6b6b;
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
