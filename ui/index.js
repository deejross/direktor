const Home = { template: "<h1>Home</h1>" }

window.app = new Vue({
    el: '#app',
    router: new VueRouter({
        routes: [
            { path: "/", component: Home }
        ]
    }),
    vuetify: new Vuetify({
        theme: { dark: true }
    }),
});
