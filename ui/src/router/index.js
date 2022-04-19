//
//  UI for IVPN Client Desktop
//  https://github.com/ivpn/desktop-app
//
//  Created by Stelnykovych Alexandr.
//  Copyright (c) 2020 Privatus Limited.
//
//  This file is part of the UI for IVPN Client Desktop.
//
//  The UI for IVPN Client Desktop is free software: you can redistribute it and/or
//  modify it under the terms of the GNU General Public License as published by the Free
//  Software Foundation, either version 3 of the License, or (at your option) any later version.
//
//  The UI for IVPN Client Desktop is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
//  or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public License for more
//  details.
//
//  You should have received a copy of the GNU General Public License
//  along with the UI for IVPN Client Desktop. If not, see <https://www.gnu.org/licenses/>.
//

import Vue from "vue";
import VueRouter from "vue-router";
import Main from "../views/Main.vue";
import AccountLimit from "../views/AccountLimit.vue";
import Settings from "../views/Settings.vue";
import Update from "../views/dialogs/Update.vue";

import DlgEapEnable from "../views/dialogs/eapEnable.vue";
import DlgEapDisable from "../views/dialogs/eapDisable.vue";

import { IpcModalDialogType } from "@/ipc/types.js";

Vue.use(VueRouter);

const mainRoutes = [
  {
    path: "/",
    name: "Main",
    component: Main,
  },
  {
    path: "/account_limit",
    name: "AccountLimit",
    component: AccountLimit,
  },
  {
    path: "/settings/:view",
    name: "settings",
    component: Settings,
  },
  {
    path: "/test",
    name: "Test",
    // route level code-splitting
    // this generates a separate chunk (about.[hash].js) for this route
    // which is lazy-loaded when the route is visited.
    component: () =>
      import(/* webpackChunkName: "about" */ "../views/Test.vue"),
  },
];
const forbiddenToChangeRouteFrom = [
  {
    path: "/update",
    name: "Update",
    component: Update,
  },
  {
    path: `/${IpcModalDialogType.EnableEAP}`,
    component: DlgEapEnable,
  },
  {
    path: `/${IpcModalDialogType.DisableEAP}`,
    component: DlgEapDisable,
  },
];

const routes = mainRoutes.concat(forbiddenToChangeRouteFrom);

const router = new VueRouter({
  mode: "hash",
  base: process.env.BASE_URL,
  routes,
});

router.beforeEach((to, from, next) => {
  // check if route allowed
  for (let route of forbiddenToChangeRouteFrom) {
    if (from.path === route.path) {
      next(false);
      return;
    }
  }
  // allow route
  next();
});

export default router;
