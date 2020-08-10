export default {
  namespaced: true,

  state: {
    serversFavoriteView: false,
    pauseConnectionTill: null //new Date()
  },

  mutations: {
    serversFavoriteView(state, value) {
      state.serversFavoriteView = value;
    },
    pauseConnectionTill(state, value) {
      state.pauseConnectionTill = value;
    }
  },

  // can be called from renderer
  actions: {
    serversFavoriteView(context, value) {
      context.commit("serversFavoriteView", value);
    },
    pauseConnectionTill(context, value) {
      context.commit("pauseConnectionTill", value);
    }
  }
};
