using System;
using System.Threading;
using System.Threading.Tasks;
using Google.Apis.Auth.OAuth2;
using Google.Cloud.PubSub.V1;
using Grpc.Auth;

namespace ExemploMesageriaCSharp
{
	class Program
	{
		static async Task<object> PullMessagesAsync(string projectId, string subscriptionId, bool acknowledge)
		{
			var credentials = GoogleCredential.FromFile("configPubsub.json");
			var settings = new SubscriberClient.ClientCreationSettings(credentials: credentials.ToChannelCredentials());

			SubscriptionName subscriptionName = new SubscriptionName(projectId, subscriptionId);
			SubscriberClient subscriber = await SubscriberClient.CreateAsync(subscriptionName, settings);

			var sub = subscriber.StartAsync(async (PubsubMessage message, CancellationToken cancel) =>
			{
				string text = message.Data.ToStringUtf8();
				await Console.Out.WriteLineAsync($"Messagem {message.MessageId}: {text}");
				
				return acknowledge ? SubscriberClient.Reply.Ack	: SubscriberClient.Reply.Nack;
			});

			/*
				Neste exemplo, roda por apenas 3 segundos. Para rodar de forma indefinida, remova o delay abaixo e use: await sub;
			*/
			//await sub; // <= Remova este comentário para rodar indefinidamente
			await Task.Delay(3000);
			await subscriber.StopAsync(CancellationToken.None); // Para o recebimento das mensagens			

			return 0;
		}

		public static string IdProduto => "Fonecido pela lifeApp";
		public static string IdProjeto => "paje-datastore";
		public static bool AceitarMensagem => true;

		static async Task Main(string[] args)
		{
			Console.WriteLine("Executando mensageria");
			await PullMessagesAsync(IdProjeto, $"{IdProduto}.pedidos", AceitarMensagem);
		}
	}
}
