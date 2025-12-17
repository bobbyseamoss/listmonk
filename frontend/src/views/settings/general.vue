<template>
  <div class="items">
    <b-field :label="$t('settings.general.siteName')" label-position="on-border">
      <b-input v-model="data['app.site_name']" name="app.site_name" :label="$t('settings.general.siteName')"
        :maxlength="300" required />
    </b-field>

    <b-field :label="$t('settings.general.rootURL')" label-position="on-border"
      :message="$t('settings.general.rootURLHelp')">
      <b-input v-model="data['app.root_url']" name="app.root_url" placeholder="https://listmonk.yoursite.com"
        :maxlength="300" required type="url" pattern="https?://.*" />
    </b-field>

    <div class="columns">
      <div class="column is-6">
        <b-field :label="$t('settings.general.logoURL')" label-position="on-border"
          :message="$t('settings.general.logoURLHelp')">
          <b-input v-model="data['app.logo_url']" name="app.logo_url" placeholder="https://listmonk.yoursite.com/logo.png"
            :maxlength="300" type="url" pattern="https?://.*" />
        </b-field>
      </div>
      <div class="column is-6">
        <b-field :label="$t('settings.general.faviconURL')" label-position="on-border"
          :message="$t('settings.general.faviconURLHelp')">
          <b-input v-model="data['app.favicon_url']" name="app.favicon_url"
            placeholder="https://listmonk.yoursite.com/favicon.png" :maxlength="300"
            type="url" pattern="https?://.*" />
        </b-field>
      </div>
    </div>

    <hr />
    <b-field :label="$t('settings.general.fromEmail')" label-position="on-border"
      :message="$t('settings.general.fromEmailHelp')">
      <b-input v-model="data['app.from_email']" name="app.from_email"
        placeholder="Listmonk <noreply@listmonk.yoursite.com>" pattern="((.+?)\s)?<(.+?)@(.+?)>" :maxlength="300" />
    </b-field>
    <b-field :label="$t('settings.general.adminNotifEmails')" label-position="on-border"
      :message="$t('settings.general.adminNotifEmailsHelp')">
      <b-taginput v-model="data['app.notify_emails']" name="app.notify_emails"
        :before-adding="(v) => v.match(/(.+?)@(.+?)/)" placeholder="you@yoursite.com" />
    </b-field>

    <hr />
    <div>
      <h2 class="is-size-4 mb-5">
        Email Headers
      </h2>
      <b-field label="Abuse Email" label-position="on-border"
        message="Email address for spam/abuse reports. Used in X-Report-Abuse header.">
        <b-input v-model="data['app.abuse_email']" name="app.abuse_email"
          placeholder="abuse@yoursite.com" type="email" :maxlength="200" />
      </b-field>

      <b-field label="Feedback Sender ID" label-position="on-border"
        message="5-15 character identifier for Google Postmaster Tools Feedback-ID header. Must be consistent across all emails.">
        <b-input v-model="data['app.feedback_sender_id']" name="app.feedback_sender_id"
          placeholder="yoursite" :maxlength="15" pattern="[a-zA-Z0-9-]{5,15}" />
      </b-field>
    </div>

    <hr />

    <div>
      <h2 class="is-size-4 mb-5">
        {{ $tc('globals.terms.subscriptions', 2) }}
      </h2>
      <div class="columns">
        <div class="column is-4">
          <b-field :label="$t('settings.general.enablePublicSubPage')"
            :message="$t('settings.general.enablePublicSubPageHelp')">
            <b-switch v-model="data['app.enable_public_subscription_page']" name="app.enable_public_subscription_page" />
          </b-field>
        </div>
        <div class="column is-4">
          <b-field :label="$t('settings.general.sendOptinConfirm')"
            :message="$t('settings.general.sendOptinConfirmHelp')">
            <b-switch v-model="data['app.send_optin_confirmation']" name="app.send_optin_confirmation" />
          </b-field>
        </div>
      </div>
    </div>
    <hr />

    <div>
      <h2 class="is-size-4 mb-5">
        {{ $t('campaigns.archive') }}
      </h2>
      <div class="columns">
        <div class="column is-4">
          <b-field :label="$t('settings.general.enablePublicArchive')"
            :message="$t('settings.general.enablePublicArchiveHelp')">
            <b-switch v-model="data['app.enable_public_archive']" name="app.enable_public_archive" />
          </b-field>
        </div>
        <div class="column is-4">
          <b-field :label="$t('settings.general.enablePublicArchiveRSSContent')"
            :message="$t('settings.general.enablePublicArchiveRSSContentHelp')">
            <b-switch v-model="data['app.enable_public_archive_rss_content']"
              name="app.enable_public_archive_rss_content" />
          </b-field>
        </div>
      </div>
    </div>

    <hr />
    <b-field :label="$t('settings.general.checkUpdates')" :message="$t('settings.general.checkUpdatesHelp')">
      <b-switch v-model="data['app.check_updates']" name="app.check_updates" />
    </b-field>

    <hr />
    <b-field :label="$t('settings.general.language')" label-position="on-border" :addons="false">
      <b-select v-model="data['app.lang']" name="app.lang">
        <option v-for="l in serverConfig.langs" :key="l.code" :value="l.code">
          {{ l.name }}
        </option>
      </b-select>
      <p class="mt-2">
        <a href="https://listmonk.app/docs/i18n/#additional-language-packs" target="_blank" rel="noopener noreferer">{{
          $t('globals.buttons.more') }} &rarr;</a>
      </p>
    </b-field>
  </div>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';

export default Vue.extend({
  props: {
    form: {
      type: Object, default: () => { },
    },
  },

  data() {
    return {
      data: this.form,
    };
  },

  computed: {
    ...mapState(['serverConfig', 'loading']),
  },

});
</script>
