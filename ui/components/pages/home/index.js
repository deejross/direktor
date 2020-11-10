export default {
    props: [
        "domains"
    ],
    methods: {

    },
    template: `
        <v-container no-gutters fill-height fluid>
            <v-navigation-drawer v-if="domains.length > 0">
                <v-list-item>
                    <v-list-item-content>
                        <v-list-item-title>Domains</v-list-item-title>
                    </v-list-item-content>
                </v-list-item>

                <v-list dense nav>
                    <v-list-item v-for="(domain, index) of domains" :key="index">
                        {{ domain.name }}
                    </v-list-item>
                </v-list>
            </v-navigation-drawer>

            <v-card v-if="domains.length === 0" align="center" justify="center" class="mx-auto">
                <v-card-title>No Domains Configured</v-card-title>
                <v-card-text>The backend service does not have any domains configured. You will need to update its configuration to use this application.</v-card-text>
            </v-card>
        </v-container>
    `
}