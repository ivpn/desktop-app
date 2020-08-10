import Vue from "vue";
import VueRouter from "vue-router";
import Main from "../views/Main.vue";
import AccountLimit from "../views/AccountLimit.vue";
import Settings from "../views/Settings.vue";

Vue.use(VueRouter);

const routes = [
  {
    path: "/",
    name: "Main",
    component: Main
  },
  {
    path: "/account_limit",
    name: "AccountLimit",
    component: AccountLimit
  },
  {
    path: "/settings/:view",
    component: Settings
  },
  {
    path: "/settings*",
    name: "settings",
    component: Settings
  },
  {
    path: "/test",
    name: "Test",
    // route level code-splitting
    // this generates a separate chunk (about.[hash].js) for this route
    // which is lazy-loaded when the route is visited.
    component: () => import(/* webpackChunkName: "about" */ "../views/Test.vue")
  }
];

const router = new VueRouter({
  mode: "hash", // "history",
  base: process.env.BASE_URL,
  routes
});

export default router;
