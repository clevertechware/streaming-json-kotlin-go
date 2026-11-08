package demo.movements

import org.springframework.context.annotation.Bean
import org.springframework.context.annotation.Configuration
import org.springframework.core.task.AsyncTaskExecutor
import org.springframework.scheduling.concurrent.ThreadPoolTaskExecutor
import org.springframework.web.servlet.config.annotation.AsyncSupportConfigurer
import org.springframework.web.servlet.config.annotation.WebMvcConfigurer

/**
 * Chaque export occupe un thread de ce pool pendant toute la durée du flux, à raison d'une connexion JDBC
 * par export : la taille du pool borne le nombre d'exports simultanés.
 */
@Configuration
class AsyncConfig : WebMvcConfigurer {

    @Bean
    fun streamingExecutor(): AsyncTaskExecutor = ThreadPoolTaskExecutor().apply {
        corePoolSize = 4
        maxPoolSize = 4
        queueCapacity = 8
        setThreadNamePrefix("export-")
    }

    override fun configureAsyncSupport(configurer: AsyncSupportConfigurer) {
        configurer.setTaskExecutor(streamingExecutor())
    }
}
